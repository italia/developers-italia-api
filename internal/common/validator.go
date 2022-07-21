package common

import (
	"errors"

	"github.com/go-playground/validator/v10"
)

type ValidationError struct {
	FailedField string `json:"field"`
	Tag         string `json:"rule"`
	Value       string `json:"providedValue"`
}

func ValidateStruct(s interface{}) []ValidationError {
	validate := validator.New()

	var validationErrors []ValidationError

	if err := validate.Struct(s); err != nil {
		var ve validator.ValidationErrors
		if ok := errors.As(err, &ve); !ok {
			return nil
		}

		for _, err := range ve {
			var element ValidationError

			element.FailedField = err.StructNamespace()
			element.Tag = err.Tag()
			element.Value = err.Param()
			validationErrors = append(validationErrors, element)
		}
	}

	return validationErrors
}
