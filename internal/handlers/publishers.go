package handlers

import (
	"errors"

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

	if err := p.db.Find(&publishers).Error; err != nil {
		return common.Error(
			fiber.StatusInternalServerError,
			"can't get Publisher",
			fiber.ErrInternalServerError.Message,
		)
	}

	return ctx.JSON(&publishers)
}

// GetPublisher gets the publisher with the given ID and returns any error encountered.
func (p *Publisher) GetPublisher(ctx *fiber.Ctx) error {
	publisher := models.Publisher{}

	if err := p.db.First(&publisher, ctx.Params("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.Error(fiber.StatusNotFound, "can't get Publisher", "Publisher was not found")
		}

		return common.Error(fiber.StatusInternalServerError, "can't get Publisher", "internal server error")
	}

	return ctx.JSON(&publisher)
}

// PostPublisher creates a new publisher.
func (p *Publisher) PostPublisher(ctx *fiber.Ctx) error {
	publisher := new(requests.Publisher)

	if err := ctx.BodyParser(&publisher); err != nil {
		return common.Error(fiber.StatusBadRequest, "can't create Publisher", "invalid json")
	}

	if err := common.ValidateStruct(*publisher); err != nil {
		return common.Error(fiber.StatusUnprocessableEntity, "can't create Publisher", "invalid format")
	}

	if err := p.db.Create(&publisher).Error; err != nil {
		return common.Error(fiber.StatusInternalServerError, "can't create Publisher", "db error")
	}

	return ctx.JSON(&publisher)
}

// PatchPublisher updates the publisher with the given ID.
func (p *Publisher) PatchPublisher(ctx *fiber.Ctx) error {
	publisherReq := new(requests.Publisher)

	if err := ctx.BodyParser(publisherReq); err != nil {
		return common.Error(fiber.StatusBadRequest, "can't update Publisher", "invalid json")
	}

	if err := common.ValidateStruct(*publisherReq); err != nil {
		return common.Error(fiber.StatusUnprocessableEntity, "can't update Publisher", "invalid format")
	}

	publisher := models.Publisher{}

	if err := p.db.First(&publisher, ctx.Params("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.Error(fiber.StatusNotFound, "can't update Publisher", "Publisher was not found")
		}

		return common.Error(fiber.StatusInternalServerError, "can't update Publisher", "internal server error")
	}

	publisher.Name = publisherReq.Name

	if err := p.db.Updates(&publisher).Error; err != nil {
		return common.Error(fiber.StatusInternalServerError, "can't update Publisher", "db error")
	}

	return ctx.JSON(&publisher)
}

// DeletePublisher deletes the publisher with the given ID.
func (p *Publisher) DeletePublisher(ctx *fiber.Ctx) error {
	var publisher models.Publisher

	if err := p.db.Delete(&publisher, ctx.Params("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.Error(fiber.StatusNotFound, "can't delete Publisher", "Publisher was not found")
		}

		return common.Error(fiber.StatusInternalServerError, "can't delete Publisher", "db error")
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}
