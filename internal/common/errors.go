package common

import (
	"errors"

	"github.com/gofiber/fiber/v2"
)

var (
	ErrAuthentication = errors.New("token authentication failed")

	ErrKeyLen = errors.New("PASETO_KEY must be 32 bytes long once base64-decoded")
)

func Error(status int, title string, detail string, extra ...any) ProblemJSONError {
	p := ProblemJSONError{Title: title, Detail: detail, Status: status}
	if extra != nil {
		p.ValidationErrors = extra
	}

	return p
}

func CustomErrorHandler(ctx *fiber.Ctx, err error) error {
	var problemJSON *ProblemJSONError

	// Retrieve the custom status code if it's a fiber.*Error
	var e *fiber.Error
	if errors.Is(err, e) {
		problemJSON = &ProblemJSONError{Status: e.Code, Title: e.Message}
	}

	if errors.Is(err, ErrAuthentication) {
		problemJSON = &ProblemJSONError{Status: fiber.StatusUnauthorized, Title: err.Error()}
	}

	if problemJSON == nil {
		//nolint:errorlint
		switch e := err.(type) {
		case ProblemJSONError:
			problemJSON = &e
		default:
			problemJSON = &ProblemJSONError{Status: fiber.StatusNotFound, Title: fiber.ErrNotFound.Message}
		}
	}

	err = ctx.Status(problemJSON.Status).JSON(problemJSON)

	// Needs to be after the call to JSON(), to override the
	// automatic Content-Type
	ctx.Set(fiber.HeaderContentType, "application/problem+json")

	return err
}
