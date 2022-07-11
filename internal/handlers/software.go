package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
	"github.com/italia/developers-italia-api/internal/common"
	"github.com/italia/developers-italia-api/internal/handlers/general"
	"github.com/italia/developers-italia-api/internal/models"
	"github.com/italia/developers-italia-api/internal/requests"
	"gorm.io/gorm"
)

type SoftwareInterface interface {
	GetAllSoftware(ctx *fiber.Ctx) error
	GetSoftware(ctx *fiber.Ctx) error
	PostSoftware(ctx *fiber.Ctx) error
	PatchSoftware(ctx *fiber.Ctx) error
	DeleteSoftware(ctx *fiber.Ctx) error
}

type Software struct {
	db *gorm.DB
}

func NewSoftware(db *gorm.DB) *Software {
	return &Software{db: db}
}

// GetAllSoftware gets the list of all software and returns any error encountered.
func (p *Software) GetAllSoftware(ctx *fiber.Ctx) error {
	var software []models.Software

	stmt := p.db.Begin().Preload("URLs")

	paginator := general.NewPaginator(ctx)

	result, cursor, err := paginator.Paginate(stmt, &software)
	if err != nil {
		return common.Error(
			fiber.StatusBadRequest,
			"can't get Software",
			"wrong cursor format in page[after] or page[before]",
		)
	}

	if result.Error != nil {
		return common.Error(
			fiber.StatusInternalServerError,
			"can't get Software",
			result.Error.Error(),
			// fiber.ErrInternalServerError.Message,
		)
	}

	return ctx.JSON(fiber.Map{"data": &software, "links": general.PaginationLinks(cursor)})
}

// GetSoftware gets the software with the given ID and returns any error encountered.
func (p *Software) GetSoftware(ctx *fiber.Ctx) error {
	software := models.Software{}

	if err := p.db.First(&software, ctx.Params("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.Error(fiber.StatusNotFound, "can't get Software", "Software was not found")
		}

		return common.Error(
			fiber.StatusInternalServerError,
			"can't get Software",
			// fiber.ErrInternalServerError.Message,
			err.Error(),
		)
	}

	return ctx.JSON(&software)
}

// PostSoftware creates a new software.
func (p *Software) PostSoftware(ctx *fiber.Ctx) error {
	softwareReq := new(requests.Software)

	if err := ctx.BodyParser(&softwareReq); err != nil {
		return common.Error(fiber.StatusBadRequest, "can't create Software", "invalid json")
	}

	if err := common.ValidateStruct(*softwareReq); err != nil {
		return common.Error(fiber.StatusUnprocessableEntity, "can't create Software", "invalid format", err)
	}

	softwareURLs := []models.SoftwareURL{}
	for _, u := range softwareReq.URLs {
		softwareURLs = append(softwareURLs, models.SoftwareURL{ID: utils.UUIDv4(), URL: u})
	}

	software := models.Software{
		ID:   utils.UUIDv4(),
		URLs: softwareURLs,
	}

	if err := p.db.Create(&software).Error; err != nil {
		return common.Error(fiber.StatusInternalServerError, "can't create Software", err.Error())
	}

	return ctx.JSON(&software)
}

// PatchSoftware updates the software with the given ID.
func (p *Software) PatchSoftware(ctx *fiber.Ctx) error {
	softwareReq := new(requests.Software)

	if err := ctx.BodyParser(softwareReq); err != nil {
		return common.Error(fiber.StatusBadRequest, "can't update Software", "invalid json")
	}

	if err := common.ValidateStruct(*softwareReq); err != nil {
		return common.Error(fiber.StatusUnprocessableEntity, "can't update Software", "invalid format", err)
	}

	software := models.Software{}

	if err := p.db.First(&software, ctx.Params("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.Error(fiber.StatusNotFound, "can't update Software", "Software was not found")
		}

		return common.Error(fiber.StatusInternalServerError, "can't update Software", "internal server error")
	}

	softwareURLs := []models.SoftwareURL{}
	for _, u := range softwareReq.URLs {
		softwareURLs = append(softwareURLs, models.SoftwareURL{ID: utils.UUIDv4(), URL: u})
	}

	software.URLs = softwareURLs

	if err := p.db.Updates(&software).Error; err != nil {
		return common.Error(fiber.StatusInternalServerError, "can't update Software", "db error")
	}

	return ctx.JSON(&software)
}

// DeleteSoftware deletes the software with the given ID.
func (p *Software) DeleteSoftware(ctx *fiber.Ctx) error {
	var software models.Software

	if err := p.db.Delete(&software, ctx.Params("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.Error(fiber.StatusNotFound, "can't delete Software", "Software was not found")
		}

		return common.Error(fiber.StatusInternalServerError, "can't delete Software", "db error")
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}
