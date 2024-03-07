package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
	"github.com/italia/developers-italia-api/internal/common"
	"github.com/italia/developers-italia-api/internal/handlers/general"
	"github.com/italia/developers-italia-api/internal/models"
	"github.com/pilagod/gorm-cursor-paginator/v2/paginator"
	"gorm.io/gorm"
)

type LogInterface interface {
	GetLogs(ctx *fiber.Ctx) error
	GetLog(ctx *fiber.Ctx) error
	PostLog(ctx *fiber.Ctx) error
	PatchLog(ctx *fiber.Ctx) error
	DeleteLog(ctx *fiber.Ctx) error

	GetSoftwareLogs(ctx *fiber.Ctx) error
	PostSoftwareLog(ctx *fiber.Ctx) error
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

	stmt, err := general.Clauses(ctx, p.db, "message")
	if err != nil {
		return common.Error(
			fiber.StatusUnprocessableEntity,
			"can't get Logs",
			err.Error(),
		)
	}

	// Logs are returned in descending order, last first
	paginator := general.NewPaginatorWithConfig(ctx, &paginator.Config{Order: paginator.DESC})

	result, cursor, err := paginator.Paginate(stmt, &logs)
	if err != nil {
		return common.Error(
			fiber.StatusUnprocessableEntity,
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

	if err := p.db.First(&log, "id = ?", ctx.Params("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.Error(fiber.StatusNotFound, "can't get Log", "Log was not found")
		}

		return common.Error(
			fiber.StatusInternalServerError,
			"can't get Log",
			fiber.ErrInternalServerError.Message,
		)
	}

	return ctx.JSON(&log)
}

// PostLog creates a new log.
func (p *Log) PostLog(ctx *fiber.Ctx) error {
	const errMsg = "can't create Log"

	logReq := new(common.Log)

	if err := common.ValidateRequestEntity(ctx, logReq, errMsg); err != nil {
		return err //nolint:wrapcheck
	}

	log := models.Log{ID: utils.UUIDv4(), Message: logReq.Message}

	if err := p.db.Create(&log).Error; err != nil {
		return common.Error(fiber.StatusInternalServerError, errMsg, "db error")
	}

	return ctx.JSON(&log)
}

// PatchLog updates the log with the given ID.
func (p *Log) PatchLog(ctx *fiber.Ctx) error {
	const errMsg = "can't update Log"

	logReq := new(common.Log)

	if err := common.ValidateRequestEntity(ctx, logReq, errMsg); err != nil {
		return err //nolint:wrapcheck
	}

	log := models.Log{}

	if err := p.db.First(&log, "id = ?", ctx.Params("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.Error(fiber.StatusNotFound, errMsg, "Log was not found")
		}

		return common.Error(fiber.StatusInternalServerError, errMsg, fiber.ErrInternalServerError.Message)
	}

	log.Message = logReq.Message

	if err := p.db.Updates(&log).Error; err != nil {
		return common.Error(fiber.StatusInternalServerError, errMsg, "db error")
	}

	return ctx.JSON(&log)
}

// DeleteLog deletes the log with the given ID.
func (p *Log) DeleteLog(ctx *fiber.Ctx) error {
	var log models.Log

	result := p.db.Delete(&log, "id = ?", ctx.Params("id"))

	if result.Error != nil {
		return common.Error(fiber.StatusInternalServerError, "can't delete Log", "db error")
	}

	if result.RowsAffected == 0 {
		return common.Error(fiber.StatusNotFound, "can't delete Log", "Log was not found")
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}

// GetSoftwareLogs gets the logs associated to a Software with the given ID and returns any error encountered.
func (p *Log) GetSoftwareLogs(ctx *fiber.Ctx) error {
	var logs []models.Log

	software := models.Software{}

	if err := p.db.First(&software, "id = ?", ctx.Params("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.Error(fiber.StatusNotFound, "can't get Software", "Software was not found")
		}

		return common.Error(
			fiber.StatusInternalServerError,
			"can't get Software",
			fiber.ErrInternalServerError.Message,
		)
	}

	stmt := p.db.
		Where(map[string]interface{}{"entity_type": models.Software{}.TableName()}).
		Where("entity_id = ?", software.ID)

	// Logs are returned in descending order, last first
	paginator := general.NewPaginatorWithConfig(ctx, &paginator.Config{Order: paginator.DESC})

	result, cursor, err := paginator.Paginate(stmt, &logs)
	if err != nil {
		return common.Error(
			fiber.StatusUnprocessableEntity,
			"can't get Software",
			"wrong cursor format in page[after] or page[before]",
		)
	}

	if result.Error != nil {
		return common.Error(
			fiber.StatusInternalServerError,
			"can't get Software",
			fiber.ErrInternalServerError.Message,
		)
	}

	return ctx.JSON(fiber.Map{"data": &logs, "links": general.PaginationLinks(cursor)})
}

// PostSoftwareLog creates a new log associated to a Software with the given ID and returns any error encountered.
func (p *Log) PostSoftwareLog(ctx *fiber.Ctx) error {
	const errMsg = "can't create Log"

	logReq := new(common.Log)

	software := models.Software{}
	if err := p.db.First(&software, "id = ?", ctx.Params("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.Error(fiber.StatusNotFound, "can't create Log", "Software was not found")
		}

		return common.Error(
			fiber.StatusInternalServerError,
			"can't get Software",
			fiber.ErrInternalServerError.Message,
		)
	}

	if err := common.ValidateRequestEntity(ctx, logReq, errMsg); err != nil {
		return err //nolint:wrapcheck
	}

	table := models.Software{}.TableName()

	log := models.Log{
		ID:         utils.UUIDv4(),
		Message:    logReq.Message,
		EntityID:   &software.ID,
		EntityType: &table,
	}

	if err := p.db.Create(&log).Error; err != nil {
		return common.Error(fiber.StatusInternalServerError, errMsg, "db error")
	}

	return ctx.JSON(&log)
}
