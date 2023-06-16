package main

import (
	"log"
	"os"
	"time"

	"github.com/caarlos0/env/v6"
	"gorm.io/gorm"

	"github.com/italia/developers-italia-api/internal/common"
	"github.com/italia/developers-italia-api/internal/database"
	"github.com/italia/developers-italia-api/internal/handlers"
	"github.com/italia/developers-italia-api/internal/middleware"
	"github.com/italia/developers-italia-api/internal/models"
	"github.com/italia/developers-italia-api/internal/webhooks"

	"github.com/ansrivas/fiberprometheus/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cache"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	app := Setup()
	if err := app.Listen(":3000"); err != nil {
		log.Fatal(err)
	}
}

func Setup() *fiber.App {
	if err := env.Parse(&common.EnvironmentConfig); err != nil {
		panic(err)
	}

	gormDB, err := database.NewDatabase(common.EnvironmentConfig.Database)
	if err != nil {
		panic(err)
	}

	// Setup a goroutine acting as a worker for events sent to the
	// EventChan channel.
	//
	// It dispatches the webhooks related to the event that occurred
	// (es. Publisher creation, Software delete, etc.)
	go func() {
		for event := range models.EventChan {
			if err := webhooks.DispatchWebhooks(event, gormDB); err != nil {
				log.Println(err)
			}
		}
	}()

	app := fiber.New(fiber.Config{
		ErrorHandler: common.CustomErrorHandler,
	})

	// Automatically recover panics in handlers
	app.Use(recover.New())

	// Use Fiber Rate API Limiter
	if !common.EnvironmentConfig.IsTest() && common.EnvironmentConfig.MaxRequests != 0 {
		app.Use(limiter.New(limiter.Config{
			Max:               common.EnvironmentConfig.MaxRequests,
			LimiterMiddleware: limiter.SlidingWindow{},
			KeyGenerator: func(ctx *fiber.Ctx) string {
				return ctx.IP() + ctx.Get(fiber.HeaderXForwardedFor, "")
			},
		}))
	}

	app.Use(cache.New(cache.Config{
		Next: func(ctx *fiber.Ctx) bool {
			// Don't cache /status
			return ctx.Route().Path == "/v1/status"
		},
		Methods:      []string{fiber.MethodGet, fiber.MethodHead},
		CacheControl: true,
		Expiration:   10 * time.Second, //nolint:gomnd
		KeyGenerator: func(ctx *fiber.Ctx) string {
			return ctx.Path() + string(ctx.Context().QueryArgs().QueryString())
		},
	}))

	if common.EnvironmentConfig.PasetoKey == nil {
		log.Printf("PASETO_KEY not set, API will run in read-only mode")

		common.EnvironmentConfig.PasetoKey = middleware.NewRandomPasetoKey()
	}

	prometheus := fiberprometheus.New(os.Args[0])
	prometheus.RegisterAt(app, "/metrics")
	app.Use(prometheus.Middleware)

	app.Use(middleware.NewPasetoMiddleware(common.EnvironmentConfig))

	setupHandlers(app, gormDB)

	return app
}

func setupHandlers(app *fiber.App, gormDB *gorm.DB) {
	publisherHandler := handlers.NewPublisher(gormDB)
	softwareHandler := handlers.NewSoftware(gormDB)
	statusHandler := handlers.NewStatus(gormDB)
	logHandler := handlers.NewLog(gormDB)
	publisherWebhookHandler := handlers.NewWebhook[models.Publisher](gormDB)
	softwareWebhookHandler := handlers.NewWebhook[models.Software](gormDB)

	//nolint:varnamelen
	v1 := app.Group("/v1")

	v1.Get("/publishers/webhooks", publisherWebhookHandler.GetResourceWebhooks)
	v1.Post("/publishers/webhooks", publisherWebhookHandler.PostResourceWebhook)
	v1.Get("/publishers/:id/webhooks", publisherWebhookHandler.GetSingleResourceWebhooks)
	v1.Post("/publishers/:id/webhooks", publisherWebhookHandler.PostSingleResourceWebhook)
	v1.Get("/publishers", publisherHandler.GetPublishers)
	v1.Get("/publishers/:id", publisherHandler.GetPublisher)
	v1.Post("/publishers", publisherHandler.PostPublisher)
	v1.Patch("/publishers/:id", publisherHandler.PatchPublisher)
	v1.Delete("/publishers/:id", publisherHandler.DeletePublisher)

	v1.Get("/software/webhooks", softwareWebhookHandler.GetResourceWebhooks)
	v1.Post("/software/webhooks", softwareWebhookHandler.PostResourceWebhook)
	v1.Get("/software/:id/webhooks", softwareWebhookHandler.GetSingleResourceWebhooks)
	v1.Post("/software/:id/webhooks", softwareWebhookHandler.PostSingleResourceWebhook)
	v1.Get("/software", softwareHandler.GetAllSoftware)
	v1.Get("/software/:id", softwareHandler.GetSoftware)
	v1.Post("/software", softwareHandler.PostSoftware)
	v1.Patch("/software/:id", softwareHandler.PatchSoftware)
	v1.Delete("/software/:id", softwareHandler.DeleteSoftware)

	v1.Get("/logs", logHandler.GetLogs)
	v1.Get("/logs/:id<guid>", logHandler.GetLog)
	v1.Post("/logs", logHandler.PostLog)
	v1.Patch("/logs/:id<guid>", logHandler.PatchLog)
	v1.Delete("/logs/:id<guid>", logHandler.DeleteLog)
	v1.Get("/software/:id/logs", logHandler.GetSoftwareLogs)
	v1.Post("/software/:id/logs", logHandler.PostSoftwareLog)

	v1.Get("/status", statusHandler.GetStatus)

	v1.Get("/webhooks/:id<guid>", publisherWebhookHandler.GetWebhook)
	v1.Patch("/webhooks/:id<guid>", publisherWebhookHandler.PatchWebhook)
	v1.Delete("/webhooks/:id<guid>", publisherWebhookHandler.DeleteWebhook)
}
