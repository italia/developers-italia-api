package main

import (
	"log"

	"github.com/caarlos0/env"

	"github.com/italia/developers-italia-api/internal/common"
	"github.com/italia/developers-italia-api/internal/handlers"

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

	publisherHandler := handlers.NewPublisher(gormDB)
	statusHandler := handlers.NewStatus(gormDB)

	app := fiber.New(fiber.Config{
		Prefork:      true,
		ErrorHandler: common.CustomErrorHandler,
	})

	// Automatically recover panics in handlers
	app.Use(recover.New())

	// Use Fiber Rate API Limiter
	app.Use(limiter.New(limiter.Config{
		Max:               envs.MaxRequests,
		LimiterMiddleware: limiter.SlidingWindow{},
	}))

	//nolint:varnamelen
	v1 := app.Group("/v1")

	v1.Get("/publishers", publisherHandler.GetPublishers)
	v1.Get("/publishers/:id", publisherHandler.GetPublisher)
	v1.Post("/publishers", publisherHandler.PostPublisher)
	v1.Patch("/publishers/:id", publisherHandler.PatchPublisher)
	v1.Delete("/publishers/:id", publisherHandler.DeletePublisher)

	app.Get("/status", statusHandler.GetStatus)

	return app
}
