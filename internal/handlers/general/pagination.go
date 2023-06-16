package general

import (
	"encoding/json"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/pilagod/gorm-cursor-paginator/v2/paginator"
)

const DefaultLimitCount = 25

var DefaultConfig = &paginator.Config{ //nolint:gochecknoglobals //can't turn it into a constant
	Keys:  []string{"CreatedAt", "ID"},
	Limit: DefaultLimitCount,
	Order: paginator.ASC,
}

type PaginationLinks paginator.Cursor

func NewPaginator(ctx *fiber.Ctx) *paginator.Paginator {
	return NewPaginatorWithConfig(ctx, DefaultConfig)
}

func NewPaginatorWithConfig(ctx *fiber.Ctx, config *paginator.Config) *paginator.Paginator {
	mergedConf := DefaultConfig

	if len(config.Keys) != 0 {
		mergedConf.Keys = config.Keys
	}

	if config.Limit != 0 {
		mergedConf.Limit = config.Limit
	}

	if config.Order != DefaultConfig.Order {
		mergedConf.Order = config.Order
	}

	paginator := paginator.New(mergedConf)

	if after := ctx.Query("page[after]"); after != "" {
		paginator.SetAfterCursor(after)
	}

	if before := ctx.Query("page[before]"); before != "" {
		paginator.SetBeforeCursor(before)
	}

	//nolint:godox // need to implement this in the future
	// TODO: make the API return the error if limit is not an integer
	size := ctx.QueryInt("page[size]", DefaultLimitCount)
	paginator.SetLimit(size)

	return paginator
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
