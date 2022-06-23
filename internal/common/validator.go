package common

import (
	"errors"

	"github.com/go-playground/validator/v10"
)

type ErrorResponse struct {
	FailedField string `json:"field"`
	Tag         string `json:"rule"`
	Value       string `json:"providedValue"`
}

func ValidateStruct(s interface{}) []*ErrorResponse {
	validate := validator.New()

	var errorResponse []*ErrorResponse

	if err := validate.Struct(s); err != nil {
		var validationErrors validator.ValidationErrors
		if ok := errors.As(err, &validationErrors); !ok {
			return nil
		}

		for _, err := range validationErrors {
			var element ErrorResponse
			element.FailedField = err.StructNamespace()
			element.Tag = err.Tag()
			element.Value = err.Param()
			errorResponse = append(errorResponse, &element)
		}
	}

	return errorResponse
}
