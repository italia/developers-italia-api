package handlers

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	db "github.com/italia/developers-italia-api/internal/database"
	"github.com/italia/developers-italia-api/internal/models"
)

func GetPublishers(c *fiber.Ctx) error {
	var publishers []models.Publisher

	db.Database.Find(&publishers)

	return c.JSON(&publishers)
}

func GetPublisher(c *fiber.Ctx) error {
	var publisher models.Publisher

	if err := db.Database.First(&publisher, c.Params("id")).Error; err != nil {
		switch err {
		case gorm.ErrRecordNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"message": err.Error(),
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": err.Error(),
			})
		}
	}

	return c.JSON(&publisher)
}

func PostPublisher(c *fiber.Ctx) error {
	var publisher models.Publisher

	if err := c.BodyParser(&publisher); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	if err := db.Database.Create(&publisher).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	return c.JSON(&publisher)
}

func PatchPublisher(c *fiber.Ctx) error {
	var publisher models.Publisher

	if err := c.BodyParser(&publisher); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": err.Error(),
		})
	}
	query := db.Database.Model(models.Publisher{}).Where("id = ?", c.Params("id"))
	if err := query.Updates(&publisher).Error; err != nil {
		switch err {
		case gorm.ErrRecordNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"message": err.Error(),
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": err.Error(),
			})
		}
	}

	return c.JSON(&publisher)
}

func DeletePublisher(c *fiber.Ctx) error {
	var publisher models.Publisher

	if err := db.Database.Delete(&publisher, c.Params("id")).Error; err != nil {
		switch err {
		case gorm.ErrRecordNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"message": err.Error(),
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": err.Error(),
			})
		}
	}

	return c.JSON(&publisher)
}
