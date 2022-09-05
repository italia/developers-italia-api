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
				Rule:  err.Tag(),
				Value: value,
			})
		}
	}

	return validationErrors
}

func ValidateRequestEntity(ctx *fiber.Ctx, request interface{}, entityName string) error {
	if err := ctx.BodyParser(request); err != nil {
		return Error(fiber.StatusBadRequest, fmt.Sprintf("can't update %s", entityName), "invalid json")
	}

	if err := ValidateStruct(request); err != nil {
		return ErrorWithValidationErrors(
			fiber.StatusUnprocessableEntity, fmt.Sprintf("can't update %s", entityName), "invalid format", err,
		)
	}

	return nil
}
