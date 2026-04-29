package general

import (
	"encoding/json"
	"maps"
	"net/url"
	"slices"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/pilagod/gorm-cursor-paginator/v2/paginator"
)

const DefaultLimitCount = 25

var DefaultConfig = &paginator.Config{ //nolint:gochecknoglobals //can't turn it into a constant
	Keys:  []string{"CreatedAt", "ID"},
	Limit: DefaultLimitCount,
	Order: paginator.ASC,
}

type PaginationLinks struct {
	prev *string
	next *string
}

func NewPaginationLinks(queries map[string]string, cursor paginator.Cursor) PaginationLinks {
	base := maps.Clone(queries)
	delete(base, "page[after]")
	delete(base, "page[before]")

	return PaginationLinks{
		prev: buildLink(base, "page[before]", cursor.Before),
		next: buildLink(base, "page[after]", cursor.After),
	}
}

func buildLink(base map[string]string, key string, cursor *string) *string {
	if cursor == nil {
		return nil
	}

	params := maps.Clone(base)
	params[key] = *cursor

	parts := make([]string, 0, len(params))
	for _, k := range slices.Sorted(maps.Keys(params)) {
		parts = append(parts, k+"="+url.QueryEscape(params[k]))
	}

	s := "?" + strings.Join(parts, "&")

	return &s
}

func NewPaginator(ctx *fiber.Ctx) *paginator.Paginator {
	return NewPaginatorWithConfig(ctx, DefaultConfig)
}

func NewPaginatorWithConfig(ctx *fiber.Ctx, config *paginator.Config) *paginator.Paginator {
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

	//nolint:godox // need to implement this in the future
	// TODO: make the API return the error if limit is not an integer
	size := ctx.QueryInt("page[size]", DefaultLimitCount)
	paginator.SetLimit(size)

	return paginator
}

func (links PaginationLinks) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Prev *string `json:"prev"`
		Next *string `json:"next"`
	}{
		Prev: links.prev,
		Next: links.next,
	})
}
