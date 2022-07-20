package main

import (
	"log"

	"github.com/caarlos0/env/v6"
	"gorm.io/gorm"

	"github.com/italia/developers-italia-api/internal/common"
	"github.com/italia/developers-italia-api/internal/handlers"
	"github.com/italia/developers-italia-api/internal/middleware"
	"github.com/italia/developers-italia-api/internal/models"
	"github.com/italia/developers-italia-api/internal/webhooks"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/italia/developers-italia-api/internal/database"
)

func main() {
	app := Setup()
	if err := app.Listen(":3000"); err != nil {
		log.Fatal(err)
	}
}

func Setup() *fiber.App {
	envs := common.Environment{}

	if err := env.Parse(&envs); err != nil {
		log.Fatal(err)
	}

	db := database.NewDatabase(envs)

	gormDB, err := db.Init(envs.Database)
	if err != nil {
		log.Fatal(err)
	}

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
	if !envs.IsTest() {
		app.Use(limiter.New(limiter.Config{
			Max:               envs.MaxRequests,
			LimiterMiddleware: limiter.SlidingWindow{},
		}))
	}

	if envs.PasetoKey == nil {
		log.Printf("PASETO_KEY not set, API will run in read-only mode")

		envs.PasetoKey = middleware.NewRandomPasetoKey()
	}

	app.Use(middleware.NewPasetoMiddleware(envs))

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
	v1.Get("/logs/:id", logHandler.GetLog)
	v1.Post("/logs", logHandler.PostLog)
	v1.Patch("/logs/:id", logHandler.PatchLog)
	v1.Delete("/logs/:id", logHandler.DeleteLog)
	v1.Get("/software/:id/logs", logHandler.GetSoftwareLogs)
	v1.Post("/software/:id/logs", logHandler.PostSoftwareLog)

	v1.Get("/status", statusHandler.GetStatus)

	v1.Get("/webhooks/:id", publisherWebhookHandler.GetWebhook)
	v1.Patch("/webhooks/:id", publisherWebhookHandler.PatchWebhook)
	v1.Delete("/webhooks/:id", publisherWebhookHandler.DeleteWebhook)
}
