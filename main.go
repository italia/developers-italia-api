package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/italia/developers-italia-api/internal/database"
	"github.com/italia/developers-italia-api/internal/handlers"
)

func main() {
	if err := database.Init(os.Getenv("DATABASE_DSN")); err != nil {
		log.Fatal(err)
	}

	app := fiber.New(fiber.Config{
		Prefork: true,
	})

	app.Get("/publishers", handlers.GetPublishers)
	app.Get("/publishers/:id", handlers.GetPublisher)
	app.Post("/publishers", handlers.PostPublisher)
	app.Patch("/publishers/:id", handlers.PatchPublisher)
	app.Delete("/publishers/:id", handlers.DeletePublisher)

	err := app.Listen(":3000")
	if err != nil {
		log.Fatal(err)
	}
}
