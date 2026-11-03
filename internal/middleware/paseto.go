package middleware

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"aidanwoods.dev/go-paseto"
	"github.com/gofiber/fiber/v2"
	"github.com/italia/open-catalog-api/internal/common"
)

var errExpiredToken = errors.New("expired paseto token")

func NewRandomPasetoKey() *common.Base64Key {
	key := make([]byte, common.SymmetricKeyLen)

	if _, err := rand.Read(key); err != nil {
		log.Fatalf("can't generate PASETO key: %s", err.Error())
	}

	return (*common.Base64Key)(key)
}

// IssueV2Token encrypts a PASETO v2.local token with the same claims the
// previous o1egl issuer wrote: iat, optional sub, and exp only when expiry
// is positive. expiry 0 means the token does not expire.
func IssueV2Token(key []byte, subject string, expiry time.Duration, now time.Time) (string, []byte, error) {
	sym, err := paseto.V2SymmetricKeyFromBytes(key)
	if err != nil {
		return "", nil, fmt.Errorf("invalid PASETO key: %w", err)
	}

	token := paseto.NewToken()
	token.SetIssuedAt(now)

	if subject != "" {
		token.SetSubject(subject)
	}

	if expiry > 0 {
		token.SetExpiration(now.Add(expiry))
	}

	encrypted := token.V2Encrypt(sym)

	return encrypted, token.ClaimsJSON(), nil
}

func NewPasetoMiddleware(envs common.Environment) fiber.Handler {
	if envs.PasetoKey == nil {
		log.Fatal("PASETO key is not set")
	}

	key, err := paseto.V2SymmetricKeyFromBytes(envs.PasetoKey[:])
	if err != nil {
		log.Fatalf("invalid PASETO key: %s", err.Error())
	}

	// NotExpired() rejects tokens that have no exp claim. token create
	// --expiry 0 and tokens already in use omit that claim, so the rule
	// below only rejects an exp that is present and in the past.
	parser := paseto.MakeParser([]paseto.Rule{rejectIfExpired})

	return func(ctx *fiber.Ctx) error {
		// Skip this authentication middleware on GET requests,
		// GETs are public.
		if ctx.Method() == fiber.MethodGet {
			return ctx.Next()
		}

		header := ctx.Get(fiber.HeaderAuthorization)

		token, ok := strings.CutPrefix(header, "Bearer ")
		if !ok || token == "" {
			return common.CustomErrorHandler(ctx, common.ErrAuthentication)
		}

		if _, err := parser.ParseV2Local(key, token); err != nil {
			return common.CustomErrorHandler(ctx, common.ErrAuthentication)
		}

		return ctx.Next()
	}
}

// rejectIfExpired rejects a token whose exp claim is present and already past.
// A missing exp stays valid. A malformed exp is rejected, which is what
// unmarshalling into o1egl's JSONToken already did.
func rejectIfExpired(token paseto.Token) error {
	if _, present := token.Claims()["exp"]; !present {
		return nil
	}

	exp, err := token.GetExpiration()
	if err != nil {
		return fmt.Errorf("invalid exp claim: %w", err)
	}

	if time.Now().After(exp) {
		return errExpiredToken
	}

	return nil
}
