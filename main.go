package main

import (
	"log"
	"os"

	"github.com/italia/developers-italia-api/internal/common"

	"github.com/gofiber/fiber/v2"
	"github.com/italia/developers-italia-api/internal/database"
	"github.com/italia/developers-italia-api/internal/handlers"
)

func main() {
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
	app.Get("/publishers", publisherHandler.GetPublishers)
	app.Get("/publishers/:id", publisherHandler.GetPublisher)
	app.Post("/publishers", publisherHandler.PostPublisher)
	app.Patch("/publishers/:id", publisherHandler.PatchPublisher)
	app.Delete("/publishers/:id", publisherHandler.DeletePublisher)

	return app
}
