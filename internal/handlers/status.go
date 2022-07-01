package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/italia/developers-italia-api/internal/common"
	"gorm.io/gorm"
)

type Status struct {
	db *gorm.DB
}

func NewStatus(db *gorm.DB) *Status {
	return &Status{db: db}
}

// GetStatus gets status of the API.
func (s *Status) GetStatus(ctx *fiber.Ctx) error {
	ctx.Append("Cache-Control", "no-cache")

	if err := s.db.Exec("SELECT 1 WHERE true").Error; err != nil {
		return common.Error(
			fiber.StatusInternalServerError,
			"can't connect to database",
			fiber.ErrInternalServerError.Message,
		)
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}
