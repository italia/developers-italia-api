package middleware

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"aidanwoods.dev/go-paseto"
	"github.com/gofiber/fiber/v2"
	"github.com/italia/open-catalog-api/internal/common"
	"github.com/stretchr/testify/require"
)

func testKey() common.Base64Key {
	var key common.Base64Key

	copy(key[:], []byte("test-paseto-key-dont-use-in-prod"))

	return key
}

func TestPasetoV2Local(t *testing.T) {
	key := testKey()
	app := fiber.New(fiber.Config{ErrorHandler: common.CustomErrorHandler})
	app.Use(NewPasetoMiddleware(common.Environment{PasetoKey: &key}))
	app.All("/v1/probe", func(ctx *fiber.Ctx) error {
		return ctx.SendString("ok")
	})

	issued, claims, err := IssueV2Token(key[:], "caller", time.Hour, time.Now().UTC())
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(issued, "v2.local."))
	require.Contains(t, string(claims), `"sub":"caller"`)
	require.Contains(t, string(claims), `"exp"`)

	never, neverClaims, err := IssueV2Token(key[:], "", 0, time.Now().UTC())
	require.NoError(t, err)
	require.NotContains(t, string(neverClaims), `"exp"`)

	expired := encryptExpired(t, key[:])

	tests := []struct {
		name   string
		method string
		auth   string
		code   int
	}{
		{name: "token issued by this binary", method: http.MethodPost, auth: "Bearer " + issued, code: http.StatusOK},
		{name: "expiry 0 has no exp and is accepted", method: http.MethodPost, auth: "Bearer " + never, code: http.StatusOK},
		{name: "expired exp is rejected", method: http.MethodPost, auth: "Bearer " + expired, code: http.StatusUnauthorized},
		{name: "token that does not decrypt", method: http.MethodPost, auth: "Bearer v2.local.invalid", code: http.StatusUnauthorized},
		{name: "missing authorization", method: http.MethodPost, auth: "", code: http.StatusUnauthorized},
		{name: "GET stays public", method: http.MethodGet, auth: "", code: http.StatusOK},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req, err := http.NewRequest(test.method, "/v1/probe", nil)
			require.NoError(t, err)

			req.Host = "localhost"
			if test.auth != "" {
				req.Header.Set(fiber.HeaderAuthorization, test.auth)
			}

			res, err := app.Test(req, -1)
			require.NoError(t, err)

			body, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			require.Equal(t, test.code, res.StatusCode, string(body))
		})
	}
}

func encryptExpired(t *testing.T, key []byte) string {
	t.Helper()

	sym, err := paseto.V2SymmetricKeyFromBytes(key)
	require.NoError(t, err)

	token := paseto.NewToken()
	token.SetIssuedAt(time.Now().Add(-2 * time.Hour))
	token.SetExpiration(time.Now().Add(-time.Hour))

	return token.V2Encrypt(sym)
}
