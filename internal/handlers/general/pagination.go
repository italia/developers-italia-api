package general

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/pilagod/gorm-cursor-paginator/v2/paginator"
)

const DefaultLimitCount = 25

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

	if size := ctx.Query("page[size]"); size != "" {
		//nolint:godox // need to implement this in the future
		// TODO: make the API return the error if limit is not an integer
		if limit, err := strconv.Atoi(size); err == nil {
			paginator.SetLimit(limit)
		}
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
