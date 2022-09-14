package handlers

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/italia/developers-italia-api/internal/database"

	"github.com/italia/developers-italia-api/internal/handlers/general"

	"github.com/gofiber/fiber/v2/utils"

	"github.com/PuerkitoBio/purell"
	normalizer "github.com/dimuska139/go-email-normalizer"
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

const normalizeFlags = purell.FlagsUsuallySafeGreedy | purell.FlagRemoveWWW

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
	normalize := normalizer.NewNormalizer()
	request := new(common.PublisherPost)

	err := common.ValidateRequestEntity(ctx, request, "can't create Publisher")
	if err != nil {
		return err //nolint:wrapcheck
	}

	normalizedEmail := normalize.Normalize(request.Email)

	publisher := &models.Publisher{
		ID:     utils.UUIDv4(),
		Email:  normalizedEmail,
		Active: request.Active,
	}

	if request.ExternalCode != "" {
		publisher.ExternalCode = &request.ExternalCode
	}

	if request.Description != "" {
		publisher.Description = &request.Description
	}

	publisher.Description = &request.Description

	for _, codeHost := range request.CodeHosting {
		urlAddress, _ := url.Parse(codeHost.URL)
		normalizedURL := purell.NormalizeURL(urlAddress, normalizeFlags)

		publisher.CodeHosting = append(publisher.CodeHosting,
			models.CodeHosting{
				ID:    utils.UUIDv4(),
				URL:   normalizedURL,
				Group: codeHost.Group,
			})
	}

	if err := p.db.Create(&publisher).Error; err != nil {
		switch database.WrapErrors(err) { //nolint:errorlint
		case common.ErrDbRecordNotFound:
			return common.Error(fiber.StatusNotFound,
				"can't create Publisher",
				"Publisher was not found")
		case common.ErrDbUniqueConstraint:
			return common.Error(fiber.StatusConflict,
				"can't create Publisher",
				"Publisher with provided description, email, external_code or CodeHosting URL already exists")
		default:
			return common.Error(fiber.StatusInternalServerError,
				"can't create Publisher",
				"internal server error")
		}
	}

	return ctx.JSON(&publisher)
}

// PatchPublisher updates the publisher with the given ID. CodeHosting URLs will be overwritten from the request.
func (p *Publisher) PatchPublisher(ctx *fiber.Ctx) error {
	requests := new(common.PublisherPatch)

	if err := common.ValidateRequestEntity(ctx, requests, "can't update Publisher"); err != nil {
		return err //nolint:wrapcheck
	}

	publisher := models.Publisher{}

	if err := p.db.Transaction(func(gormTrx *gorm.DB) error {
		return p.updatePublisherTrx(gormTrx, publisher, ctx, requests)
	}); err != nil {
		return err //nolint:wrapcheck
	}

	return ctx.JSON(&publisher)
}

func (p *Publisher) updatePublisherTrx(
	gormTrx *gorm.DB,
	publisher models.Publisher,
	ctx *fiber.Ctx,
	publisherReq *common.PublisherPatch,
) error {
	if err := gormTrx.Model(&models.Publisher{}).Preload("CodeHosting").
		First(&publisher, "id = ?", ctx.Params("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.Error(fiber.StatusNotFound, "Not found", "can't update Publisher. Publisher was not found")
		}

		return common.Error(fiber.StatusInternalServerError,
			"can't update Publisher",
			fmt.Errorf("db error: %w", err).Error())
	}

	if publisherReq.Description != "" {
		publisher.Description = &publisherReq.Description
	}

	if publisherReq.Email != "" {
		publisher.Email = publisherReq.Email
	}

	if publisherReq.ExternalCode != "" {
		publisher.ExternalCode = &publisherReq.ExternalCode
	}

	if publisherReq.CodeHosting != nil && len(publisherReq.CodeHosting) > 0 {
		gormTrx.Delete(&publisher.CodeHosting)

		for _, URLAddress := range publisherReq.CodeHosting {
			publisher.CodeHosting = append(publisher.CodeHosting, models.CodeHosting{ID: utils.UUIDv4(), URL: URLAddress.URL})
		}
	}

	if err := p.db.Updates(&publisher).Error; err != nil {
		return common.Error(fiber.StatusInternalServerError,
			"can't update Publisher",
			fmt.Errorf("db error: %w", err).Error())
	}

	return nil
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
