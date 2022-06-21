package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	db "github.com/italia/developers-italia-api/internal/database"
	"github.com/italia/developers-italia-api/internal/models"
)

func GetPublishers(ctx *fiber.Ctx) error {
	var publishers []models.Publisher

	db.Database.Find(&publishers)

	return ctx.JSON(&publishers)
}

func GetPublisher(ctx *fiber.Ctx) error {
	var publisher models.Publisher

	if err := db.Database.First(&publisher, ctx.Params("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"message": err.Error(),
			})
		}

		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	return ctx.JSON(&publisher)
}

func PostPublisher(ctx *fiber.Ctx) error {
	var publisher models.Publisher

	if err := ctx.BodyParser(&publisher); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	if err := db.Database.Create(&publisher).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	return ctx.JSON(&publisher)
}

func PatchPublisher(ctx *fiber.Ctx) error {
	var publisher models.Publisher

	if err := ctx.BodyParser(&publisher); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	query := db.Database.Model(models.Publisher{}).Where("id = ?", ctx.Params("id"))

	if err := query.Updates(&publisher).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"message": err.Error(),
			})
		}

		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	return ctx.JSON(&publisher)
}

func DeletePublisher(ctx *fiber.Ctx) error {
	var publisher models.Publisher

	if err := db.Database.Delete(&publisher, ctx.Params("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"message": err.Error(),
			})
		}

		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	return ctx.JSON(&publisher)
}
