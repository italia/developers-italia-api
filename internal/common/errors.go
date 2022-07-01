package common

import (
	"errors"

	"github.com/gofiber/fiber/v2"
)

func Error(status int, title string, detail string) ProblemJSONError {
	return ProblemJSONError{Title: title, Detail: detail, Status: status}
}

func ValidationError(ctx *fiber.Ctx, errors []*ErrorResponse) error {
	return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"error": errors,
	})
}

func CustomErrorHandler(ctx *fiber.Ctx, err error) error {
	var problemJSON ProblemJSONError

	// Retrieve the custom status code if it's a fiber.*Error
	var e *fiber.Error
	if ok := errors.Is(err, e); ok {
		problemJSON = ProblemJSONError{Status: e.Code, Title: e.Message}
	}

	//nolint:errorlint
	switch e := err.(type) {
	case ProblemJSONError:
		problemJSON = e
	default:
		problemJSON = ProblemJSONError{Status: fiber.StatusNotFound, Title: fiber.ErrNotFound.Message}
	}

	err = ctx.Status(problemJSON.Status).JSON(problemJSON)

	// Needs to be after the call to JSON(), to override the
	// automatic Content-Type
	ctx.Set(fiber.HeaderContentType, "application/problem+json")

	return err
}
