package common

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestApp() *fiber.App {
	return fiber.New(fiber.Config{ErrorHandler: CustomErrorHandler})
}

func TestCustomErrorHandler_FiberError(t *testing.T) {
	app := newTestApp()
	app.Get("/bad-input", func(ctx *fiber.Ctx) error {
		return fiber.NewError(fiber.StatusBadRequest, "bad input")
	})

	req, err := http.NewRequest(http.MethodGet, "/bad-input", nil)
	require.NoError(t, err)
	req.Host = "localhost"

	res, err := app.Test(req, -1)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusBadRequest, res.StatusCode)

	var body ProblemJSONError
	require.NoError(t, json.NewDecoder(res.Body).Decode(&body))

	assert.Equal(t, "bad input", body.Title)
	assert.Equal(t, fiber.StatusBadRequest, body.Status)
}

func TestCustomErrorHandler_ProblemJSON(t *testing.T) {
	app := newTestApp()
	app.Get("/problem", func(ctx *fiber.Ctx) error {
		return Error(fiber.StatusUnprocessableEntity, "validation failed", "field x is missing")
	})

	req, err := http.NewRequest(http.MethodGet, "/problem", nil)
	require.NoError(t, err)
	req.Host = "localhost"

	res, err := app.Test(req, -1)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusUnprocessableEntity, res.StatusCode)

	var body ProblemJSONError
	require.NoError(t, json.NewDecoder(res.Body).Decode(&body))

	assert.Equal(t, "validation failed", body.Title)
}

func TestCustomErrorHandler_AuthError(t *testing.T) {
	app := newTestApp()
	app.Get("/auth", func(ctx *fiber.Ctx) error {
		return ErrAuthentication
	})

	req, err := http.NewRequest(http.MethodGet, "/auth", nil)
	require.NoError(t, err)
	req.Host = "localhost"

	res, err := app.Test(req, -1)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusUnauthorized, res.StatusCode)
}
