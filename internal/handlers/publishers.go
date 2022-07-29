package handlers

import (
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v2/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/italia/developers-italia-api/internal/common"
	"github.com/italia/developers-italia-api/internal/models"
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

	stmt := p.db

	if all := ctx.Query("all", ""); all == "" {
		stmt = stmt.Scopes(models.Active)
	}

	if err := stmt.Find(&publishers).Error; err != nil {
		return common.Error(
			fiber.StatusInternalServerError,
			"can't get Publisher",
			fiber.ErrInternalServerError.Message,
		)
	}

	return ctx.JSON(common.NewResponse(publishers))
}

// GetPublisher gets the publisher with the given ID and returns any error encountered.
func (p *Publisher) GetPublisher(ctx *fiber.Ctx) error {
	publisher := models.Publisher{}

	if err := p.db.First(&publisher, "id = ?", ctx.Params("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.Error(fiber.StatusNotFound, "can't get Publisher", "Publisher was not found")
		}

		return common.Error(fiber.StatusInternalServerError, "can't get Publisher", "internal server error")
	}

	return ctx.JSON(&publisher)
}

// PostPublisher creates a new publisher.
func (p *Publisher) PostPublisher(ctx *fiber.Ctx) error {
	request := common.Publisher{}

	if err := ctx.BodyParser(&request); err != nil {
		return common.Error(fiber.StatusBadRequest, "can't create Publisher", "invalid json")
	}

	if err := common.ValidateStruct(&request); err != nil {
		return common.ErrorWithValidationErrors(
			fiber.StatusUnprocessableEntity, "can't create Publisher", "invalid format", err,
		)
	}

	publisher := &models.Publisher{
		ID:    utils.UUIDv4(),
		Email: request.Email,
	}

	for _, URLAddress := range request.CodeHosting {
		publisher.CodeHosting = append(publisher.CodeHosting, models.CodeHosting{URL: URLAddress.URL})
	}

	if err := p.db.Create(&publisher).Error; err != nil {
		return common.Error(fiber.StatusInternalServerError, "can't create Publisher", "db error")
	}

	return ctx.JSON(common.NewResponse(publisher))
}

// PatchPublisher updates the publisher with the given ID.
func (p *Publisher) PatchPublisher(ctx *fiber.Ctx) error {
	publisherReq := new(common.Publisher)

	if err := ctx.BodyParser(publisherReq); err != nil {
		return common.Error(fiber.StatusBadRequest, "can't update Publisher", "invalid json")
	}

	if err := common.ValidateStruct(*publisherReq); err != nil {
		return common.ErrorWithValidationErrors(
			fiber.StatusUnprocessableEntity, "can't update Publisher", "invalid format", err,
		)
	}

	publisher := models.Publisher{}

	if err := p.updatePublisher(ctx, publisher, publisherReq); err != nil {
		return err
	}

	return ctx.JSON(&publisher)
}

func (p *Publisher) updatePublisher(ctx *fiber.Ctx, publisher models.Publisher, req *common.Publisher) error {
	err := p.db.Transaction(func(gormTrx *gorm.DB) error {
		if err := gormTrx.Model(&models.Publisher{}).
			Preload("CodeHosting").
			First(&publisher, "id = ?", ctx.Params("id")).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return common.Error(fiber.StatusNotFound, "can't update Publisher", "Publisher was not found")
			}

			return common.Error(fiber.StatusInternalServerError, "can't update Publisher", fiber.ErrInternalServerError.Message)
		}

		gormTrx.Delete(&publisher.CodeHosting)

		for _, URLAddress := range req.CodeHosting {
			publisher.CodeHosting = append(publisher.CodeHosting, models.CodeHosting{URL: URLAddress.URL})
		}

		if err := p.db.Updates(&publisher).Error; err != nil {
			return common.Error(fiber.StatusUnprocessableEntity, "can't update Publisher", err.Error())
		}

		return nil
	})

	return fmt.Errorf("update publisher error: %w", err)
}

// DeletePublisher deletes the publisher with the given ID.
func (p *Publisher) DeletePublisher(ctx *fiber.Ctx) error {
	var publisher models.Publisher

	if err := p.db.Delete(&publisher, "id = ?", ctx.Params("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.Error(fiber.StatusNotFound, "can't delete Publisher", "Publisher was not found")
		}

		return common.Error(fiber.StatusInternalServerError, "can't delete Publisher", "db error")
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}
