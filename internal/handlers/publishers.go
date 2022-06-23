package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/italia/developers-italia-api/internal/common"
	db "github.com/italia/developers-italia-api/internal/database"
	"github.com/italia/developers-italia-api/internal/models"
	"github.com/italia/developers-italia-api/internal/requests"
)

func GetPublishers(ctx *fiber.Ctx) error {
	var publishers []models.Publisher

	db.Database.Find(&publishers)

	return ctx.JSON(&publishers)
}

func GetPublisher(ctx *fiber.Ctx) error {
	publisher := models.Publisher{}

	if err := db.Database.First(&publisher, ctx.Params("id")).Error; err != nil {
		return common.ServerError(ctx, err) //nolint:wrapcheck
	}

	return ctx.JSON(&publisher)
}

func PostPublisher(ctx *fiber.Ctx) error {
	publisher := new(models.Publisher)

	if err := ctx.BodyParser(publisher); err != nil {
		return common.UnprocessableEntity(ctx) //nolint:wrapcheck
	}

	if err := common.ValidateStruct(*publisher); err != nil {
		return common.ValidationError(ctx, err) //nolint:wrapcheck
	}

	if err := db.Database.Create(&publisher).Error; err != nil {
		return common.ServerError(ctx, err) //nolint:wrapcheck
	}

	return ctx.JSON(&publisher)
}

func PatchPublisher(ctx *fiber.Ctx) error {
	publisherReq := new(requests.Publisher)

	if err := ctx.BodyParser(publisherReq); err != nil {
		return common.UnprocessableEntity(ctx) //nolint:wrapcheck
	}

	if err := common.ValidateStruct(*publisherReq); err != nil {
		return common.ValidationError(ctx, err) //nolint:wrapcheck
	}

	publisher := models.Publisher{}

	if err := db.Database.First(&publisher, ctx.Params("id")).Error; err != nil {
		return err
	}

	publisher.Name = publisherReq.Name

	if err := db.Database.Updates(&publisher).Error; err != nil {
		return common.ServerError(ctx, err) //nolint:wrapcheck
	}

	return ctx.JSON(&publisher)
}

func DeletePublisher(ctx *fiber.Ctx) error {
	var publisher models.Publisher

	requestID := ctx.Params("id")

	if err := db.Database.First(&publisher, requestID).Error; err != nil {
		return err
	}

	if err := db.Database.Delete(&publisher, requestID).Error; err != nil {
		return common.ServerError(ctx, err) //nolint:wrapcheck
	}

	return ctx.JSON(&publisher)
}
