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
	Value string `json:"value,omitempty"`
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
		switch validationError.Field {
			case "codeHosting":
				errors = append(errors, "codeHosting is required and should contain at least an 'url' object")
			case "url":
				errors = append(errors, "url is not a valid url")
			case "alternativeId":
				errors = append(errors, "alternativeId does not respect its size limits (1-255)")
			case "description":
				errors = append(errors, "description is required")
			case "email":
				errors = append(errors, "email is not a valid email")
			case "aliases":
				errors = append(errors, "aliases must be an array of urls")
			case "message": 
				errors = append(errors, "message is required")
			default:
				errors = append(errors, validationError.Field + " is not valid")
		}
	}

	errorDetails := fmt.Sprintf("invalid format: %s", strings.Join(errors, ", "))

	return errorDetails
}