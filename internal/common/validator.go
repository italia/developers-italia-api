package common

import (
	"errors"
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
		var errors []string

		for _, validationError := range err {
			switch validationError.Field {
				case "url":
					errors = append(errors, "url is not a valid url")
					break
				case "alternativeId":
					errors = append(errors, "alternativeId does not respect its size limit (max 255 characters, min 1 character)")
					break
				default:
					errors = append(errors, validationError.Field + " is not valid")
					break
			}
		}

		errorDetails := strings.Join(errors, ", ")
		
		return ErrorWithValidationErrors(
			fiber.StatusUnprocessableEntity, errorMessage, errorDetails, err,
		)
	}

	return nil
}