package general

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/pilagod/gorm-cursor-paginator/v2/paginator"
)

const (
	DefaultLimitCount = 25
	MaxLimitCount     = 100
)

var DefaultConfig = &paginator.Config{ //nolint:gochecknoglobals //can't turn it into a constant
	Keys:  []string{"CreatedAt", "ID"},
	Limit: DefaultLimitCount,
	Order: paginator.ASC,
}

var (
	errInvalidPageSize    = errors.New("page[size] must be an integer")
	errPageSizeOutOfRange = fmt.Errorf("page[size] must be between 1 and %d", MaxLimitCount)
)

type PaginationLinks paginator.Cursor

func NewPaginator(ctx *fiber.Ctx) (*paginator.Paginator, error) {
	return NewPaginatorWithConfig(ctx, DefaultConfig)
}

func NewPaginatorWithConfig(ctx *fiber.Ctx, config *paginator.Config) (*paginator.Paginator, error) {
	mergedConf := *DefaultConfig

	if len(config.Keys) != 0 {
		mergedConf.Keys = config.Keys
	}

	if config.Limit != 0 {
		mergedConf.Limit = config.Limit
	}

	if config.Order != "" {
		mergedConf.Order = config.Order
	}

	paginator := paginator.New(&mergedConf)

	if after := ctx.Query("page[after]"); after != "" {
		paginator.SetAfterCursor(after)
	}

	if before := ctx.Query("page[before]"); before != "" {
		paginator.SetBeforeCursor(before)
	}

	size, err := pageSizeFromQuery(ctx)
	if err != nil {
		return nil, err
	}

	paginator.SetLimit(size)

	return paginator, nil
}

func pageSizeFromQuery(ctx *fiber.Ctx) (int, error) {
	raw := ctx.Query("page[size]")
	if raw == "" {
		return DefaultLimitCount, nil
	}

	size, err := strconv.Atoi(raw)
	if err != nil {
		return 0, errInvalidPageSize
	}

	if size < 1 || size > MaxLimitCount {
		return 0, errPageSizeOutOfRange
	}

	return size, nil
}

func createLink(cursor *string, field string) *string {
	if cursor != nil {
		res := fmt.Sprintf("?page[%s]=%s", field, *cursor)

		return &res
	}

	return nil
}

func (links PaginationLinks) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Prev *string `json:"prev"`
		Next *string `json:"next"`
	}{
		Prev: createLink(links.Before, "before"),
		Next: createLink(links.After, "after"),
	})
}
