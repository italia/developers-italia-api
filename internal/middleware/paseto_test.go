package middleware

import (
	"encoding/base64"
	"net/http/httptest"
	"testing"
	"time"

	"aidanwoods.dev/go-paseto"
	"github.com/gofiber/fiber/v2"
	"github.com/italia/open-catalog-api/internal/common"
	"github.com/stretchr/testify/require"
)

// Test key used by the test suite, a base64-encoded 32 bytes secret.
const (
	testPasetoKey       = "dGVzdC1wYXNldG8ta2V5LWRvbnQtdXNlLWluLXByb2Q="
	testPasetoGoodToken = "Bearer v2.local.TwwHUQEi8hr2Eo881_Bs5vK9dHOR5BgEU24QRf-U7VmUwI1yOEA6mFT0EsXioMkFT_T-jjrtIJ_Nv8f6hR6ifJXUOuzWEkm9Ijq1mqSjQatD3aDqKMyjjBA"
)

func testEnvironment(t *testing.T) common.Environment {
	t.Helper()

	var key common.Base64Key

	require.NoError(t, key.UnmarshalText([]byte(testPasetoKey)))

	return common.Environment{PasetoKey: &key}
}

func newTestApp(t *testing.T) *fiber.App {
	t.Helper()

	app := fiber.New(fiber.Config{
		ErrorHandler: common.CustomErrorHandler,
	})

	app.Use(NewPasetoMiddleware(testEnvironment(t)))
	app.Get("/resource", func(ctx *fiber.Ctx) error { return ctx.SendStatus(fiber.StatusOK) })
	app.Post("/resource", func(ctx *fiber.Ctx) error { return ctx.SendStatus(fiber.StatusOK) })

	return app
}

func doRequest(t *testing.T, app *fiber.App, method, authorization string) int {
	t.Helper()

	req := httptest.NewRequest(method, "/resource", nil)

	if authorization != "" {
		req.Header.Set(fiber.HeaderAuthorization, authorization)
	}

	resp, err := app.Test(req)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	return resp.StatusCode
}

func TestPasetoMiddleware_ValidToken(t *testing.T) {
	app := newTestApp(t)

	require.Equal(t, fiber.StatusOK, doRequest(t, app, fiber.MethodPost, testPasetoGoodToken))
}

func TestPasetoMiddleware_GeneratedToken(t *testing.T) {
	app := newTestApp(t)

	rawKey, err := base64.StdEncoding.DecodeString(testPasetoKey)
	require.NoError(t, err)

	key, err := paseto.V2SymmetricKeyFromBytes(rawKey)
	require.NoError(t, err)

	token := paseto.NewToken()
	token.SetIssuedAt(time.Now())
	token.SetExpiration(time.Now().Add(time.Hour))

	require.Equal(t,
		fiber.StatusOK,
		doRequest(t, app, fiber.MethodPost, pasetoTokenPrefix+token.V2Encrypt(key)),
	)
}

func TestPasetoMiddleware_GetIsPublic(t *testing.T) {
	app := newTestApp(t)

	require.Equal(t, fiber.StatusOK, doRequest(t, app, fiber.MethodGet, ""))
}

func TestPasetoMiddleware_MissingToken(t *testing.T) {
	app := newTestApp(t)

	require.Equal(t, fiber.StatusUnauthorized, doRequest(t, app, fiber.MethodPost, ""))
}

func TestPasetoMiddleware_WrongPrefix(t *testing.T) {
	app := newTestApp(t)

	require.Equal(t,
		fiber.StatusUnauthorized,
		doRequest(t, app, fiber.MethodPost, "Token v2.local.whatever"),
	)
}

func TestPasetoMiddleware_InvalidToken(t *testing.T) {
	app := newTestApp(t)

	require.Equal(t,
		fiber.StatusUnauthorized,
		doRequest(t, app, fiber.MethodPost, "Bearer v2.local.invalid"),
	)
}

func TestPasetoMiddleware_TokenWithoutExpiry(t *testing.T) {
	app := newTestApp(t)

	rawKey, err := base64.StdEncoding.DecodeString(testPasetoKey)
	require.NoError(t, err)

	key, err := paseto.V2SymmetricKeyFromBytes(rawKey)
	require.NoError(t, err)

	// Tokens issued by older releases carry no "exp" claim and must keep working.
	token := paseto.NewToken()
	token.SetIssuedAt(time.Now().Add(-2 * time.Hour))

	require.Equal(t,
		fiber.StatusOK,
		doRequest(t, app, fiber.MethodPost, pasetoTokenPrefix+token.V2Encrypt(key)),
	)
}

func TestPasetoMiddleware_TokenSignedWithAnotherKey(t *testing.T) {
	app := newTestApp(t)

	otherKey := paseto.NewV2SymmetricKey()

	token := paseto.NewToken()
	token.SetIssuedAt(time.Now())
	token.SetExpiration(time.Now().Add(time.Hour))

	require.Equal(t,
		fiber.StatusUnauthorized,
		doRequest(t, app, fiber.MethodPost, pasetoTokenPrefix+token.V2Encrypt(otherKey)),
	)
}
