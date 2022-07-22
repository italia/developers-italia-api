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

// type WebhookInterface interface {
// 	GetWebhooks(ctx *fiber.Ctx) error
// 	GetWebhook(ctx *fiber.Ctx) error
// 	PostWebhook(ctx *fiber.Ctx) error
// 	PatchWebhook(ctx *fiber.Ctx) error
// 	DeleteWebhook(ctx *fiber.Ctx) error

// 	GetSoftwareWebhooks(ctx *fiber.Ctx) error
// 	PostSoftwareWebhook(ctx *fiber.Ctx) error
// }

type Webhook[T models.Model] struct {
	db *gorm.DB
}

func NewWebhook[T models.Model](db *gorm.DB) *Webhook[T] {
	return &Webhook[T]{db: db}
}

// // GetWebhooks gets the list of all webhooks and returns any error encountered.
// func (p *Webhook) GetWebhooks[T Model](ctx *fiber.Ctx) error {
// 	var webhooks []models.Webhook

// 	stmt := p.db.Begin()

// 	stmt, err := general.Clauses(ctx, stmt, "")
// 	if err != nil {
// 		return common.Error(
// 			fiber.StatusUnprocessableEntity,
// 			"can't get Webhooks",
// 			err.Error(),
// 		)
// 	}

// 	paginator := general.NewPaginator(ctx)

// 	result, cursor, err := paginator.Paginate(stmt, &webhooks)
// 	if err != nil {
// 		return common.Error(
// 			fiber.StatusUnprocessableEntity,
// 			"can't get Webhooks",
// 			"wrong cursor format in page[after] or page[before]",
// 		)
// 	}

// 	if result.Error != nil {
// 		return common.Error(
// 			fiber.StatusInternalServerError,
// 			"can't get Webhooks",
// 			fiber.ErrInternalServerError.Message,
// 		)
// 	}

// 	return ctx.JSON(fiber.Map{"data": &webhooks, "links": general.PaginationLinks(cursor)})
// }

// // GetWebhook gets the webhook with the given ID and returns any error encountered.
// func (p *Webhook) GetWebhook(ctx *fiber.Ctx) error {
// 	webhook := models.Webhook{}

// 	if err := p.db.First(&webhook, "id = ?", ctx.Params("id")).Error; err != nil {
// 		if errors.Is(err, gorm.ErrRecordNotFound) {
// 			return common.Error(fiber.StatusNotFound, "can't get Webhook", "Webhook was not found")
// 		}

// 		return common.Error(fiber.StatusInternalServerError, "can't get Webhook", "internal server error")
// 	}

// 	return ctx.JSON(&webhook)
// }

// // PostWebhook creates a new webhook.
// func (p *Webhook) PostWebhook(ctx *fiber.Ctx) error {
// 	webhookReq := new(common.Webhook)

// 	if err := ctx.BodyParser(&webhookReq); err != nil {
// 		return common.Error(fiber.StatusBadRequest, "can't create Webhook", "invalid json")
// 	}

// 	if err := common.ValidateStruct(*webhookReq); err != nil {
// 		return common.ErrorWithValidationErrors(fiber.StatusUnprocessableEntity, "can't create Webhook", "invalid format", err)
// 	}

// 	webhook := models.Webhook{ID: utils.UUIDv4(), Message: webhookReq.Message}

// 	if err := p.db.Create(&webhook).Error; err != nil {
// 		return common.Error(fiber.StatusInternalServerError, "can't create Webhook", "db error")
// 	}

// 	return ctx.JSON(&webhook)
// }

// // PatchWebhook updates the webhook with the given ID.
// func (p *Webhook) PatchWebhook(ctx *fiber.Ctx) error {
// 	webhookReq := new(common.Webhook)

// 	if err := ctx.BodyParser(webhookReq); err != nil {
// 		return common.Error(fiber.StatusBadRequest, "can't update Webhook", "invalid json")
// 	}

// 	if err := common.ValidateStruct(*webhookReq); err != nil {
// 		return common.ErrorWithValidationErrors(fiber.StatusUnprocessableEntity, "can't update Webhook", "invalid format", err)
// 	}

// 	webhook := models.Webhook{}

// 	if err := p.db.First(&webhook, "id = ?", ctx.Params("id")).Error; err != nil {
// 		if errors.Is(err, gorm.ErrRecordNotFound) {
// 			return common.Error(fiber.StatusNotFound, "can't update Webhook", "Webhook was not found")
// 		}

// 		return common.Error(fiber.StatusInternalServerError, "can't update Webhook", "internal server error")
// 	}

// 	webhook.Message = webhookReq.Message

// 	if err := p.db.Updates(&webhook).Error; err != nil {
// 		return common.Error(fiber.StatusInternalServerError, "can't update Webhook", "db error")
// 	}

// 	return ctx.JSON(&webhook)
// }

// // DeleteWebhook deletes the webhook with the given ID.
// func (p *Webhook) DeleteWebhook(ctx *fiber.Ctx) error {
// 	var webhook models.Webhook

// 	result := p.db.Delete(&webhook, "id = ?", ctx.Params("id"))

// 	if result.Error != nil {
// 		return common.Error(fiber.StatusInternalServerError, "can't delete Webhook", "db error")
// 	}

// 	if result.RowsAffected == 0 {
// 		return common.Error(fiber.StatusNotFound, "can't delete Webhook", "Webhook was not found")
// 	}

// 	return ctx.SendStatus(fiber.StatusNoContent)
// }

// GetResourceWebhooks gets the webhooks associated to resources (fe. Publishers)
// and returns any error encountered.
func (p *Webhook[T]) GetResourceWebhooks(ctx *fiber.Ctx) error {
	var webhooks []models.Webhook

	webhooks = append(webhooks, models.Webhook{})

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

// GetSingleResourceWebhooks gets the webhooks associated to a resource with the given ID and returns any error encountered.
func (p *Webhook[T]) GetSingleResourceWebhooks(ctx *fiber.Ctx) error {
	var webhooks []models.Webhook

	webhooks = append(webhooks, models.Webhook{})

	var resource T

	if err := p.db.First(&resource, "id = ?", ctx.Params("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.Error(fiber.StatusNotFound, "can't get resource", "resource was not found")
		}

		return common.Error(
			fiber.StatusInternalServerError,
			"can't get Webhooks",
			fiber.ErrInternalServerError.Message,
		)
	}

	stmt := p.db.
		Where(map[string]interface{}{"entity_type": resource.TableName()}).
		Where("entity_id = ?", resource.Uuid())

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

// PostResourceWebhook creates a new webhook associated to all Software and
// returns any error encountered.
func (p *Webhook[T]) PostResourceWebhook(ctx *fiber.Ctx) error {
	webhookReq := new(common.Webhook)

	var resource T

	if err := ctx.BodyParser(&webhookReq); err != nil {
		return common.Error(fiber.StatusBadRequest, "can't create Webhook", "invalid json")
	}

	if err := common.ValidateStruct(*webhookReq); err != nil {
		return common.ErrorWithValidationErrors(fiber.StatusUnprocessableEntity, "can't create Webhook", "invalid format", err)
	}

	webhook := models.Webhook{
		ID:         utils.UUIDv4(),
		URL:        webhookReq.URL,
		Secret:     webhookReq.Secret,
		EntityType: resource.TableName(),
	}

	if err := p.db.Create(&webhook).Error; err != nil {
		return common.Error(fiber.StatusInternalServerError, "can't create Webhook", "db error")
	}

	return ctx.JSON(&webhook)
}
