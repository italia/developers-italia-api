package handlers

import (
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v2/utils"

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
	request := requests.Publisher{}

	if err := ctx.BodyParser(&request); err != nil {
		return common.Error(fiber.StatusBadRequest, "can't create Publisher", "invalid json", err)
	}

	if err := common.ValidateStruct(&request); err != nil {
		return common.Error(fiber.StatusUnprocessableEntity, "can't create Publisher", "invalid json", err)
	}

	publisher := &models.Publisher{
		ID: utils.UUID(),
		URLAddresses: []models.URLAddresses{
			{URL: request.URL},
		},
	}

	if err := p.db.Create(&publisher).Error; err != nil {
		return common.Error(fiber.StatusInternalServerError, "can't create Publisher", "db error", err)
	}

	return ctx.JSON(&publisher)
}

// PatchPublisher updates the publisher with the given ID.
func (p *Publisher) PatchPublisher(ctx *fiber.Ctx) error {
	publisherReq := new(requests.PublisherUpdate)

	if err := ctx.BodyParser(publisherReq); err != nil {
		return common.Error(fiber.StatusBadRequest, "can't update Publisher", "invalid json")
	}

	if err := common.ValidateStruct(*publisherReq); err != nil {
		return common.Error(fiber.StatusUnprocessableEntity, "can't update Publisher", "invalid format", err)
	}

	publisher := models.Publisher{}

	if err := p.updatePublisher(ctx, publisher, publisherReq); err != nil {
		return err
	}

	return ctx.JSON(&publisher)
}

func (p *Publisher) updatePublisher(ctx *fiber.Ctx, publisher models.Publisher, req *requests.PublisherUpdate) error {
	err := p.db.Transaction(func(gormTrx *gorm.DB) error {
		if err := gormTrx.Model(&models.Publisher{}).
			Preload("URLAddresses").
			First(&publisher, ctx.Params("id")).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return common.Error(fiber.StatusNotFound, "can't update Publisher", "Publisher was not found")
			}

			return common.Error(fiber.StatusInternalServerError, "can't update Publisher", fiber.ErrInternalServerError.Message)
		}

		gormTrx.Delete(&publisher.URLAddresses)

		for _, URLAddress := range req.URLAddresses {
			publisher.URLAddresses = append(publisher.URLAddresses, models.URLAddresses{URL: URLAddress.URL})
		}

		if err := p.db.Updates(&publisher).Error; err != nil {
			return common.Error(fiber.StatusInternalServerError, "can't update Publisher", "db error")
		}

		return nil
	})

	return fmt.Errorf("transaction err: %w", err)
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
