package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
	"github.com/italia/developers-italia-api/internal/common"
	"github.com/italia/developers-italia-api/internal/handlers/general"
	"github.com/italia/developers-italia-api/internal/models"
	"gorm.io/gorm"
)

type Webhook[T models.Model] struct {
	db *gorm.DB
}

func NewWebhook[T models.Model](db *gorm.DB) *Webhook[T] {
	return &Webhook[T]{db: db}
}

// GetWebhook gets the webhook with the given ID and returns any error encountered.
func (p *Webhook[T]) GetWebhook(ctx *fiber.Ctx) error {
	webhook := models.Webhook{}

	if err := p.db.First(&webhook, "id = ?", ctx.Params("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.Error(fiber.StatusNotFound, "can't get Webhook", "Webhook was not found")
		}

		return common.Error(fiber.StatusInternalServerError, "can't get Webhook", "internal server error")
	}

	return ctx.JSON(&webhook)
}

// GetResourceWebhooks gets the webhooks associated to resources
// (fe. Software, Publishers) and returns any error encountered.
func (p *Webhook[T]) GetResourceWebhooks(ctx *fiber.Ctx) error {
	var webhooks []models.Webhook

	var resource T

	stmt := p.db.Where(map[string]interface{}{"entity_type": resource.TableName()})

	paginator := general.NewPaginator(ctx)

	result, cursor, err := paginator.Paginate(stmt, &webhooks)
	if err != nil {
		return common.Error(
			fiber.StatusUnprocessableEntity,
			"can't get Webhooks",
			"wrong cursor format in page[after] or page[before]",
		)
	}

	if result.Error != nil {
		return common.Error(
			fiber.StatusInternalServerError,
			"can't get Webhooks",
			fiber.ErrInternalServerError.Message,
		)
	}

	return ctx.JSON(fiber.Map{"data": &webhooks, "links": general.PaginationLinks(cursor)})
}

// GetSingleResourceWebhooks gets the webhooks associated to a resource
// (fe. a specific Software or Publisher) with the given ID and returns any
// error encountered.
func (p *Webhook[T]) GetSingleResourceWebhooks(ctx *fiber.Ctx) error {
	var webhooks []models.Webhook

	var resource T

	if err := p.db.First(&resource, "id = ?", ctx.Params("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.Error(fiber.StatusNotFound, "can't find resource", "resource was not found")
		}

		return common.Error(
			fiber.StatusInternalServerError,
			"can't get Webhooks",
			fiber.ErrInternalServerError.Message,
		)
	}

	stmt := p.db.
		Where(map[string]interface{}{"entity_type": resource.TableName()}).
		Where("entity_id = ?", resource.UUID())

	paginator := general.NewPaginator(ctx)

	result, cursor, err := paginator.Paginate(stmt, &webhooks)
	if err != nil {
		return common.Error(
			fiber.StatusUnprocessableEntity,
			"can't get Webhooks",
			"wrong cursor format in page[after] or page[before]",
		)
	}

	if result.Error != nil {
		return common.Error(
			fiber.StatusInternalServerError,
			"can't get Webhooks",
			fiber.ErrInternalServerError.Message,
		)
	}

	return ctx.JSON(fiber.Map{"data": &webhooks, "links": general.PaginationLinks(cursor)})
}

// PostSingleResourceWebhook creates a new webhook associated to resources
// (fe. Software, Publishers) and returns any error encountered.
func (p *Webhook[T]) PostResourceWebhook(ctx *fiber.Ctx) error {
	const errMsg = "can't create Webhook"

	webhookReq := new(common.Webhook)

	var resource T

	if err := common.ValidateRequestEntity(ctx, webhookReq, errMsg); err != nil {
		return err //nolint:wrapcheck
	}

	webhook := models.Webhook{
		ID:         utils.UUIDv4(),
		URL:        webhookReq.URL,
		Secret:     webhookReq.Secret,
		EntityID:   "", // this webhook is triggered for all the resources of this kind
		EntityType: resource.TableName(),
	}

	if err := p.db.Create(&webhook).Error; err != nil {
		return common.Error(fiber.StatusInternalServerError, errMsg, "db error")
	}

	return ctx.JSON(&webhook)
}

// PostResourceWebhook creates a new webhook associated to a resource with the given ID
// (fe. a specific Software or Publisher) and returns any error encountered.
func (p *Webhook[T]) PostSingleResourceWebhook(ctx *fiber.Ctx) error {
	const errMsg = "can't create Webhook"

	webhookReq := new(common.Webhook)

	var resource T

	if err := p.db.First(&resource, "id = ?", ctx.Params("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.Error(fiber.StatusNotFound, "can't find resource", "resource was not found")
		}

		return common.Error(
			fiber.StatusInternalServerError,
			"can't get Webhooks",
			fiber.ErrInternalServerError.Message,
		)
	}

	if err := common.ValidateRequestEntity(ctx, webhookReq, errMsg); err != nil {
		return err //nolint:wrapcheck
	}

	webhook := models.Webhook{
		ID:         utils.UUIDv4(),
		URL:        webhookReq.URL,
		Secret:     webhookReq.Secret,
		EntityID:   resource.UUID(),
		EntityType: resource.TableName(),
	}

	if err := p.db.Create(&webhook).Error; err != nil {
		return common.Error(fiber.StatusInternalServerError, errMsg, "db error")
	}

	return ctx.JSON(&webhook)
}

// PatchWebhook updates the webhook with the given ID.
func (p *Webhook[T]) PatchWebhook(ctx *fiber.Ctx) error {
	const errMsg = "can't update Webhook"

	webhookReq := new(common.Webhook)

	if err := common.ValidateRequestEntity(ctx, webhookReq, errMsg); err != nil {
		return err //nolint:wrapcheck
	}

	webhook := models.Webhook{}

	if err := p.db.First(&webhook, "id = ?", ctx.Params("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.Error(fiber.StatusNotFound, errMsg, "Webhook was not found")
		}

		return common.Error(
			fiber.StatusInternalServerError,
			errMsg,
			fiber.ErrInternalServerError.Message,
		)
	}

	webhook.URL = webhookReq.URL

	if err := p.db.Updates(&webhook).Error; err != nil {
		return common.Error(fiber.StatusInternalServerError, errMsg, "db error")
	}

	return ctx.JSON(&webhook)
}

// DeleteWebhook deletes the webhook with the given ID.
func (p *Webhook[T]) DeleteWebhook(ctx *fiber.Ctx) error {
	var webhook models.Webhook

	result := p.db.Delete(&webhook, "id = ?", ctx.Params("id"))

	if result.Error != nil {
		return common.Error(fiber.StatusInternalServerError, "can't delete Webhook", "db error")
	}

	if result.RowsAffected == 0 {
		return common.Error(fiber.StatusNotFound, "can't delete Webhook", "Webhook was not found")
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}
