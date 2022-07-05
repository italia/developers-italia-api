package common

import (
	"github.com/gofiber/fiber/v2"
)

func Error(status int, title string, detail string, extra ...any) ProblemJSONError {
	p := ProblemJSONError{Title: title, Detail: detail, Status: status}
	if extra != nil {
		p.Extra = extra
	}

	return p
}

func CustomErrorHandler(ctx *fiber.Ctx, err error) error {
	var problemJSON ProblemJSONError

	//nolint:errorlint
	switch errType := err.(type) {
	case *fiber.Error:
		problemJSON = ProblemJSONError{Status: errType.Code, Title: errType.Message}
	case ProblemJSONError:
		problemJSON = errType
	default:
		problemJSON = ProblemJSONError{
			Status: fiber.StatusInternalServerError,
			Title:  fiber.ErrInternalServerError.Error(),
			Detail: errType.Error(),
		}
	}

	err = ctx.Status(problemJSON.Status).JSON(problemJSON)

	// Needs to be after the call to JSON(), to override the
	// automatic Content-Type
	ctx.Set(fiber.HeaderContentType, "application/problem+json")

	return err
}
