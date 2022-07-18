package general

import (
	"encoding/json"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/pilagod/gorm-cursor-paginator/v2/paginator"
)

const DefaultLimitCount = 10

type PaginationLinks paginator.Cursor

func NewPaginator(ctx *fiber.Ctx) *paginator.Paginator {
	paginator := paginator.New(&paginator.Config{
		Keys:  []string{"ID", "CreatedAt"},
		Limit: DefaultLimitCount,
		Order: paginator.ASC,
	})

	if after := ctx.Query("page[after]"); after != "" {
		paginator.SetAfterCursor(after)
	}

	if before := ctx.Query("page[before]"); before != "" {
		paginator.SetBeforeCursor(before)
	}

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
