package middleware

import (
	"crypto/rand"
	"log"
	"strings"

	"aidanwoods.dev/go-paseto"
	"github.com/gofiber/fiber/v2"
	"github.com/italia/open-catalog-api/internal/common"
)

const (
	// pasetoTokenPrefix is the prefix expected in the Authorization header.
	pasetoTokenPrefix = "Bearer "
	// pasetoContextKey is the fiber locals key holding the parsed token.
	pasetoContextKey = "paseto"
)

func NewRandomPasetoKey() *common.Base64Key {
	key := make([]byte, common.SymmetricKeyLen)

	if _, err := rand.Read(key); err != nil {
		log.Fatalf("can't generate PASETO key: %s", err.Error())
	}

	return (*common.Base64Key)(key)
}

func NewPasetoMiddleware(envs common.Environment) fiber.Handler {
	if envs.PasetoKey == nil {
		log.Fatalf("can't load PASETO key: key is not set")
	}

	tokenKey, err := paseto.V2SymmetricKeyFromBytes(envs.PasetoKey[:])
	if err != nil {
		log.Fatalf("can't load PASETO key: %s", err.Error())
	}

	// Tokens issued before this migration do not carry an "exp" claim,
	// so the expiry check is disabled to keep them valid.
	parser := paseto.NewParserWithoutExpiryCheck()

	return func(ctx *fiber.Ctx) error {
		// Skip this authentication middleware on GET requests,
		// GETs are public.
		if ctx.Method() == fiber.MethodGet {
			return ctx.Next()
		}

		header := ctx.Get(fiber.HeaderAuthorization)
		if header == "" {
			return common.CustomErrorHandler(ctx, common.ErrAuthentication)
		}

		tainted, found := strings.CutPrefix(header, pasetoTokenPrefix)
		if !found {
			return common.CustomErrorHandler(ctx, common.ErrAuthentication)
		}

		token, err := parser.ParseV2Local(tokenKey, tainted)
		if err != nil {
			return common.CustomErrorHandler(ctx, common.ErrAuthentication)
		}

		ctx.Locals(pasetoContextKey, token)

		return ctx.Next()
	}
}
