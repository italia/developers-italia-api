package main

import (
	"log"
	"os"

	"github.com/italia/developers-italia-api/internal/common"
	"github.com/italia/developers-italia-api/internal/handlers"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/caarlos0/env"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/italia/developers-italia-api/internal/database"
)

type Environment struct {
	MaxRequests int `env:"MAX_REQUESTS" envDefault:"20"`
}

func main() {
	environment := Environment{}
	if err := env.Parse(&environment); err != nil {
		log.Fatal(err)
	}

	app := Setup()
	if err := app.Listen(":3000"); err != nil {
		log.Fatal(err)
	}
}

func Setup() *fiber.App {
	db, err := database.Init(os.Getenv("DATABASE_DSN"))
	if err != nil {
		log.Fatal(err)
	}

	publisherHandler := handlers.NewPublisher(db)

	app := fiber.New(fiber.Config{
		Prefork:      true,
		ErrorHandler: common.CustomErrorHandler,
	})

	// Automatically recover panics in handlers
	app.Use(recover.New())

	// Use Fiber Rate API Limiter
	app.Use(limiter.New(limiter.Config{
		Max:               environment.MaxRequests,
		LimiterMiddleware: limiter.SlidingWindow{},
	}))

	app.Get("/publishers", publisherHandler.GetPublishers)
	app.Get("/publishers/:id", publisherHandler.GetPublisher)
	app.Post("/publishers", publisherHandler.PostPublisher)
	app.Patch("/publishers/:id", publisherHandler.PatchPublisher)
	app.Delete("/publishers/:id", publisherHandler.DeletePublisher)

	return app
}
