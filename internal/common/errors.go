package common

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func ServerError(ctx *fiber.Ctx, err error) error {
	return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"message": err.Error(),
	})
}

func UnprocessableEntity(ctx *fiber.Ctx) error {
	return ctx.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
		"message": "Invalid payload provided",
	})
}

func ClientErrorNotFound(ctx *fiber.Ctx, err error) error {
	return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
		"error": err.Error(),
	})
}

func ValidationError(ctx *fiber.Ctx, errors []*ErrorResponse) error {
	return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"error": errors,
	})
}

func CustomErrorHandler(ctx *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ClientErrorNotFound(ctx, err)
	}

	// Retrieve the custom status code if it's a fiber.*Error
	var e *fiber.Error
	if ok := errors.Is(err, e); ok {
		code = e.Code
	}

	return ctx.Status(code).JSON(fiber.Map{
		"message": err.Error(),
	})
}
