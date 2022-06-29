package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/italia/developers-italia-api/internal/common"
	"github.com/italia/developers-italia-api/internal/models"
	"github.com/italia/developers-italia-api/internal/requests"
	"gorm.io/gorm"
)

type PublisherInterface interface {
	GetPublishers(ctx *fiber.Ctx) error
	GetPublisher(ctx *fiber.Ctx) error
	PostPublisher(ctx *fiber.Ctx) error
	PatchPublisher(ctx *fiber.Ctx) error
	DeletePublisher(ctx *fiber.Ctx) error
}

type Publisher struct {
	db *gorm.DB
}

func NewPublisher(db *gorm.DB) *Publisher {
	return &Publisher{db: db}
}

// GetPublishers gets the list of all publishers and returns any error encountered.
func (p *Publisher) GetPublishers(ctx *fiber.Ctx) error {
	var publishers []models.Publisher

	p.db.Find(&publishers)

	return ctx.JSON(&publishers)
}

// GetPublisher gets the publisher with the given ID and returns any error encountered.
func (p *Publisher) GetPublisher(ctx *fiber.Ctx) error {
	publisher := models.Publisher{}

	if err := p.db.First(&publisher, ctx.Params("id")).Error; err != nil {
		return common.ServerError(ctx, err) //nolint:wrapcheck
	}

	return ctx.JSON(&publisher)
}

// PostPublisher creates a new publisher.
func (p *Publisher) PostPublisher(ctx *fiber.Ctx) error {
	publisher := new(models.Publisher)

	if err := ctx.BodyParser(publisher); err != nil {
		return common.UnprocessableEntity(ctx) //nolint:wrapcheck
	}

	if err := common.ValidateStruct(*publisher); err != nil {
		return common.ValidationError(ctx, err) //nolint:wrapcheck
	}

	if err := p.db.Create(&publisher).Error; err != nil {
		return common.ServerError(ctx, err) //nolint:wrapcheck
	}

	return ctx.JSON(&publisher)
}

// PatchPublisher updates the publisher with the given ID.
func (p *Publisher) PatchPublisher(ctx *fiber.Ctx) error {
	publisherReq := new(requests.Publisher)

	if err := ctx.BodyParser(publisherReq); err != nil {
		return common.UnprocessableEntity(ctx) //nolint:wrapcheck
	}

	if err := common.ValidateStruct(*publisherReq); err != nil {
		return common.ValidationError(ctx, err) //nolint:wrapcheck
	}

	publisher := models.Publisher{}

	if err := p.db.First(&publisher, ctx.Params("id")).Error; err != nil {
		return err
	}

	publisher.Name = publisherReq.Name

	if err := p.db.Updates(&publisher).Error; err != nil {
		return common.ServerError(ctx, err) //nolint:wrapcheck
	}

	return ctx.JSON(&publisher)
}

// DeletePublisher deletes the publisher with the given ID.
func (p *Publisher) DeletePublisher(ctx *fiber.Ctx) error {
	var publisher models.Publisher

	requestID := ctx.Params("id")

	if err := p.db.First(&publisher, requestID).Error; err != nil {
		return err
	}

	if err := p.db.Delete(&publisher, requestID).Error; err != nil {
		return common.ServerError(ctx, err) //nolint:wrapcheck
	}

	return ctx.JSON(&publisher)
}
