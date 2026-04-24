package general

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strconv"
	"strings"

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
		return 0, errors.New("page[size] must be an integer")
	}

	if size < 1 || size > MaxLimitCount {
		return 0, fmt.Errorf("page[size] must be between 1 and %d", MaxLimitCount)
	}

	return size, nil
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
