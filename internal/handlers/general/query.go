package general

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/italia/developers-italia-api/internal/common"
	"gorm.io/gorm"
)

func Clauses(ctx *fiber.Ctx, stmt *gorm.DB, searchFieldName string) (*gorm.DB, error) {
	ret := stmt

	if searchFieldName != "" {
		filter := ctx.Query("filter", "")

		if filter != "" {
			ret = stmt.Where(map[string]interface{}{searchFieldName: filter})
		}
	}

	if from := ctx.Query("from", ""); from != "" {
		at, err := time.Parse("2006-01-02T15:04:05Z", from)
		if err != nil {
			return nil, common.ErrInvalidDateTime
		}

		ret = stmt.Where("created_at > ?", at)
	}

	if to := ctx.Query("to", ""); to != "" {
		at, err := time.Parse("2006-01-02T15:04:05Z", to)
		if err != nil {
			return nil, common.ErrInvalidDateTime
		}

		ret = stmt.Where("created_at < ?", at)
	}

	if search := ctx.Query("search", ""); search != "" {
		ret = stmt.Where("message LIKE ?", "%"+search+"%")
	}

	return ret, nil
}
