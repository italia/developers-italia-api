package common

import (
	"errors"

	"github.com/gofiber/fiber/v2"
)

var (
	ErrAuthentication  = errors.New("token authentication failed")
	ErrInvalidDateTime = errors.New("invalid date time format (RFC 3339 needed)")
	ErrKeyLen          = errors.New("PASETO_KEY must be 32 bytes long once base64-decoded")
)

func InternalServerError(title string) ProblemJSONError {
	return Error(fiber.StatusInternalServerError, title, fiber.ErrInternalServerError.Message)
}

func Error(status int, title string, detail string) ProblemJSONError {
	return ProblemJSONError{Title: title, Detail: detail, Status: status}
}

func ErrorWithValidationErrors(
	status int, title string, validationErrors []ValidationError,
) ProblemJSONError {
	detail := GenerateErrorDetails(validationErrors)

	return ProblemJSONError{Title: title, Detail: detail, Status: status, ValidationErrors: validationErrors}
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
			problemJSON = &ProblemJSONError{Status: fiber.StatusNotFound, Title: fiber.ErrNotFound.Message, Detail: e.Error()}
		}
	}

	ctx.Status(problemJSON.Status)

	return ctx.JSON(fiber.Map{
		"title":  problemJSON.Title,
		"detail": problemJSON.Detail,
		"status": problemJSON.Status,
	}, "application/problem+json")
}
