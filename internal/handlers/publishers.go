package handlers

import (
	"errors"

	"github.com/italia/developers-italia-api/internal/handlers/general"

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

	stmt := p.db.Preload("CodeHosting")

	if all := ctx.Query("all", ""); all == "" {
		stmt = stmt.Scopes(models.Active)
	}

	paginator := general.NewPaginator(ctx)

	result, cursor, err := paginator.Paginate(stmt, &publishers)
	if err != nil {
		return common.Error(
			fiber.StatusUnprocessableEntity,
			"can't get Publishers",
			"wrong cursor format in page[after] or page[before]",
		)
	}

	if result.Error != nil {
		return common.Error(
			fiber.StatusInternalServerError,
			"can't get Publisher",
			fiber.ErrInternalServerError.Message,
		)
	}

	return ctx.JSON(fiber.Map{"data": &publishers, "links": general.PaginationLinks(cursor)})
}

// GetPublisher gets the publisher with the given ID and returns any error encountered.
func (p *Publisher) GetPublisher(ctx *fiber.Ctx) error {
	publisher := models.Publisher{}

	if err := p.db.Preload("CodeHosting").First(&publisher, "id = ?", ctx.Params("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.Error(fiber.StatusNotFound, "can't get Publisher", "Publisher was not found")
		}

		return common.Error(fiber.StatusInternalServerError, "can't get Publisher", "internal server error")
	}

	return ctx.JSON(&publisher)
}

// PostPublisher creates a new publisher.
func (p *Publisher) PostPublisher(ctx *fiber.Ctx) error {
	publisherReq := new(common.Publisher)

	err := common.ValidateRequestEntity(ctx, publisherReq, "Publisher")
	if err != nil {
		return err
	}

	publisher := &models.Publisher{
		ID:    utils.UUIDv4(),
		Email: publisherReq.Email,
	}

	if publisherReq.ExternalCode != "" {
		publisher.ExternalCode = publisherReq.ExternalCode
	}

	if publisherReq.Description != "" {
		publisher.Description = publisherReq.Description
	}

	for _, URLAddress := range publisherReq.CodeHosting {
		publisher.CodeHosting = append(publisher.CodeHosting, models.CodeHosting{ID: utils.UUIDv4(), URL: URLAddress.URL})
	}

	if err := p.db.Create(&publisher).Error; err != nil {
		return common.Error(fiber.StatusInternalServerError, "can't create Publisher", "db error")
	}

	return ctx.JSON(&publisher)
}

// PatchPublisher updates the publisher with the given ID. Please note that codeHosting URLs will be overwritten from the request.
func (p *Publisher) PatchPublisher(ctx *fiber.Ctx) error {
	publisherReq := new(common.Publisher)

	err := common.ValidateRequestEntity(ctx, publisherReq, "Publisher")
	if err != nil {
		return err
	}

	publisher := models.Publisher{}

	err = p.db.Transaction(func(gormTrx *gorm.DB) error {
		if err := gormTrx.Model(&models.Publisher{}).Preload("CodeHosting").
			First(&publisher, "id = ?", ctx.Params("id")).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return common.Error(fiber.StatusNotFound, "can't update Publisher", "Publisher was not found")
			}

			return err
		}

		if publisherReq.Description != "" {
			publisher.Description = publisherReq.Description
		}

		if publisherReq.Email != "" {
			publisher.Email = publisherReq.Email
		}

		if publisherReq.ExternalCode != "" {
			publisher.ExternalCode = publisherReq.ExternalCode
		}

		if publisherReq.CodeHosting != nil && len(publisherReq.CodeHosting) > 0 {
			gormTrx.Delete(&publisher.CodeHosting)

			for _, URLAddress := range publisherReq.CodeHosting {
				publisher.CodeHosting = append(publisher.CodeHosting, models.CodeHosting{ID: utils.UUIDv4(), URL: URLAddress.URL})
			}
		}

		return p.db.Updates(&publisher).Error
	})

	if err != nil {
		return common.Error(fiber.StatusUnprocessableEntity, "can't update Publisher", err.Error())
	}

	return ctx.JSON(&publisher)
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
