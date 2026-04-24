package general

import (
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/pilagod/gorm-cursor-paginator/v2/paginator"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

func newTestCtx() (*fiber.App, *fiber.Ctx) {
	app := fiber.New()
	ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
	return app, ctx
}

// TestNewPaginatorWithConfig_DoesNotMutateDefaultConfig verifies that calling
// NewPaginatorWithConfig with a custom Order does not mutate the global
// DefaultConfig, so subsequent callers still get the default ASC order.
func TestNewPaginatorWithConfig_DoesNotMutateDefaultConfig(t *testing.T) {
	app, ctx := newTestCtx()
	defer app.ReleaseCtx(ctx)

	_, err := NewPaginatorWithConfig(ctx, &paginator.Config{Order: paginator.DESC})
	assert.NoError(t, err)

	assert.Equal(t, paginator.ASC, DefaultConfig.Order,
		"DefaultConfig.Order must not be mutated by NewPaginatorWithConfig")
}
