package common

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/go-playground/validator/v10"
)

const (
	tagPosition      = 2
	maxProvidedValue = 255
)

type ValidationError struct {
	Field string `json:"field"`
	Rule  string `json:"rule"`
	Value string `json:"value"`
}

func ValidateStruct(validateStruct interface{}) []ValidationError {
	validate := validator.New()
	// https://github.com/go-playground/validator/issues/258#issuecomment-257281334
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		return strings.SplitN(fld.Tag.Get("json"), ",", tagPosition)[0]
	})

	var validationErrors []ValidationError

	if err := validate.Struct(validateStruct); err != nil {
		var ve validator.ValidationErrors
		if ok := errors.As(err, &ve); !ok {
			return nil
		}

		for _, err := range ve {
			var value string
			value, ok := err.Value().(string)

			if !ok {
				value = ""
			}

			valueRunes := []rune(value)
			if len(valueRunes) > maxProvidedValue {
				value = string(valueRunes[:maxProvidedValue])
			}

			validationErrors = append(validationErrors, ValidationError{
				Field: err.Field(),
				Rule:  err.Tag(),
				Value: value,
			})
		}
	}

	return validationErrors
}

func ValidateRequestEntity(ctx *fiber.Ctx, request interface{}, errorMessage string) error {
	if err := ctx.BodyParser(request); err != nil {
		return Error(fiber.StatusBadRequest, errorMessage, "invalid json")
	}

	if err := ValidateStruct(request); err != nil {
		return ErrorWithValidationErrors(
			fiber.StatusUnprocessableEntity, errorMessage, err,
		)
	}

	return nil
}

func GenerateErrorDetails(validationErrors []ValidationError) string {
	var errors []string

	for _, validationError := range validationErrors {
		switch validationError.Rule {
		case "required":
			errors = append(errors, fmt.Sprintf("%s is required", validationError.Field))
		case "email":
			errors = append(errors, fmt.Sprintf("%s is not a valid email", validationError.Field))
		case "min":
			errors = append(errors, fmt.Sprintf("%s does not meet its size limits (too short)", validationError.Field))
		case "max":
			errors = append(errors, fmt.Sprintf("%s does not meet its size limits (too long)", validationError.Field))
		case "gt":
			errors = append(errors, fmt.Sprintf("%s does not meet its size limits (too few items)", validationError.Field))
		default:
			errors = append(errors, fmt.Sprintf("%s is invalid", validationError.Field))
		}
	}

	errorDetails := fmt.Sprintf("invalid format: %s", strings.Join(errors, ", "))

	return errorDetails
}
