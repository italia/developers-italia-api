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

	if err := database.Init(os.Getenv("DATABASE_DSN")); err != nil {
		log.Fatal(err)
	}

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

	app.Get("/publishers", handlers.GetPublishers)
	app.Get("/publishers/:id", handlers.GetPublisher)
	app.Post("/publishers", handlers.PostPublisher)
	app.Patch("/publishers/:id", handlers.PatchPublisher)
	app.Delete("/publishers/:id", handlers.DeletePublisher)

	if err := app.Listen(":3000"); err != nil {
		log.Fatal(err)
	}
}
