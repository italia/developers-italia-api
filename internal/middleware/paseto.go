package middleware

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"

	"github.com/italia/developers-italia-api/internal/common"

	pasetoware "github.com/gofiber/contrib/paseto"
	"github.com/gofiber/fiber/v2"
	"github.com/o1egl/paseto"
)

func NewRandomPasetoKey() *common.Base64Key {
	key := make([]byte, common.SymmetricKeyLen)

	if _, err := rand.Read(key); err != nil {
		log.Fatalf("can't generate PASETO key: %s", err.Error())
	}

	return (*common.Base64Key)(key)
}

func NewPasetoMiddleware(envs common.Environment) fiber.Handler {
	return pasetoware.New(pasetoware.Config{
		TokenPrefix:  "Bearer",
		SymmetricKey: envs.PasetoKey[:],
		Next: func(ctx *fiber.Ctx) bool {
			// Skip this authentication middleware on GET requests,
			// GETs are public.
			return ctx.Method() == fiber.MethodGet
		},
		Validate: func(data []byte) (interface{}, error) {
			var payload paseto.JSONToken
			if err := json.Unmarshal(data, &payload); err != nil {
				return nil, fmt.Errorf("can't unmarshal PASETO token: %w", err)
			}

			return payload, nil
		},
		ErrorHandler: func(ctx *fiber.Ctx, err error) error {
			//nolint:wrapcheck // we don't want to wrap the error, we just call the error
			//                    handler with the correct error
			return common.CustomErrorHandler(ctx, common.ErrAuthentication)
		},
	})
}
