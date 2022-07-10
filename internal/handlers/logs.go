//nolint:dupl
package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/italia/developers-italia-api/internal/common"
	"github.com/italia/developers-italia-api/internal/handlers/general"
	"github.com/italia/developers-italia-api/internal/models"
	"github.com/italia/developers-italia-api/internal/requests"
	"gorm.io/gorm"
)

type LogInterface interface {
	GetLogs(ctx *fiber.Ctx) error
	GetLog(ctx *fiber.Ctx) error
	PostLog(ctx *fiber.Ctx) error
	PatchLog(ctx *fiber.Ctx) error
	DeleteLog(ctx *fiber.Ctx) error
}

type Log struct {
	db *gorm.DB
}

func NewLog(db *gorm.DB) *Log {
	return &Log{db: db}
}

// GetLogs gets the list of all logs and returns any error encountered.
func (p *Log) GetLogs(ctx *fiber.Ctx) error {
	var logs []models.Log

	stmt := p.db.Begin()

	stmt, err := general.Clauses(ctx, stmt, "message")
	if err != nil {
		return common.Error(
			fiber.StatusBadRequest,
			"can't get Logs",
			err.Error(),
		)
	}

	paginator := general.NewPaginator(ctx)

	result, cursor, err := paginator.Paginate(stmt, &logs)
	if err != nil {
		return common.Error(
			fiber.StatusBadRequest,
			"can't get Logs",
			"wrong cursor format in page[after] or page[before]",
		)
	}

	if result.Error != nil {
		return common.Error(
			fiber.StatusInternalServerError,
			"can't get Logs",
			fiber.ErrInternalServerError.Message,
		)
	}

	return ctx.JSON(fiber.Map{"data": &logs, "links": general.PaginationLinks(cursor)})
}

// GetLog gets the log with the given ID and returns any error encountered.
func (p *Log) GetLog(ctx *fiber.Ctx) error {
	log := models.Log{}

	if err := p.db.First(&log, ctx.Params("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.Error(fiber.StatusNotFound, "can't get Log", "Log was not found")
		}

		return common.Error(fiber.StatusInternalServerError, "can't get Log", "internal server error")
	}

	return ctx.JSON(&log)
}

// PostLog creates a new log.
func (p *Log) PostLog(ctx *fiber.Ctx) error {
	logReq := new(requests.Log)

	if err := ctx.BodyParser(&logReq); err != nil {
		return common.Error(fiber.StatusBadRequest, "can't create Log", "invalid json")
	}

	if err := common.ValidateStruct(*logReq); err != nil {
		return common.Error(fiber.StatusUnprocessableEntity, "can't create Log", "invalid format")
	}

	log := models.Log{Message: logReq.Message}

	if err := p.db.Create(&log).Error; err != nil {
		return common.Error(fiber.StatusInternalServerError, "can't create Log", "db error")
	}

	return ctx.JSON(&log)
}

// PatchLog updates the log with the given ID.
func (p *Log) PatchLog(ctx *fiber.Ctx) error {
	logReq := new(requests.Log)

	if err := ctx.BodyParser(logReq); err != nil {
		return common.Error(fiber.StatusBadRequest, "can't update Log", "invalid json")
	}

	if err := common.ValidateStruct(*logReq); err != nil {
		return common.Error(fiber.StatusUnprocessableEntity, "can't update Log", "invalid format")
	}

	log := models.Log{}

	if err := p.db.First(&log, ctx.Params("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.Error(fiber.StatusNotFound, "can't update Log", "Log was not found")
		}

		return common.Error(fiber.StatusInternalServerError, "can't update Log", "internal server error")
	}

	log.Message = logReq.Message

	if err := p.db.Updates(&log).Error; err != nil {
		return common.Error(fiber.StatusInternalServerError, "can't update Log", "db error")
	}

	return ctx.JSON(&log)
}

// DeleteLog deletes the log with the given ID.
func (p *Log) DeleteLog(ctx *fiber.Ctx) error {
	var log models.Log

	if err := p.db.Delete(&log, ctx.Params("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.Error(fiber.StatusNotFound, "can't delete Log", "Log was not found")
		}

		return common.Error(fiber.StatusInternalServerError, "can't delete Log", "db error")
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}
