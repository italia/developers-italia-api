package common

import (
	"errors"
	"net/url"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/italia/developers-italia-api/internal/jsondecoder"
)

// hostValidator runs the `fqdn` tag against the host of a candidate URL.
// Kept as a separate instance so we don't recurse into the struct-level
// validator we are already running.
//
//nolint:gochecknoglobals // shared validator instance for the host check
var hostValidator = validator.New()

const (
	tagPosition      = 2
	maxProvidedValue = 255
)

type ValidationError struct {
	Field string `json:"field"`
	Rule  string `json:"rule"`
	Value string `json:"value"`
}

func ValidateStruct(validateStruct any) []ValidationError {
	validate := validator.New()
	// https://github.com/go-playground/validator/issues/258#issuecomment-257281334
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		return strings.SplitN(fld.Tag.Get("json"), ",", tagPosition)[0]
	})

	_ = validate.RegisterValidation("code_hosting_url", validateCodeHostingURL)

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

func ValidateRequestEntity(ctx *fiber.Ctx, request any, errorMessage string) error {
	if err := ctx.BodyParser(request); err != nil {
		if errors.Is(err, jsondecoder.ErrUnknownField) {
			return Error(fiber.StatusUnprocessableEntity, errorMessage, err.Error())
		}

		return Error(fiber.StatusBadRequest, errorMessage, "invalid or malformed JSON")
	}

	if err := ValidateStruct(request); err != nil {
		return ErrorWithValidationErrors(
			fiber.StatusUnprocessableEntity, errorMessage, err,
		)
	}

	return nil
}

// validateCodeHostingURL rejects publisher CodeHosting URLs that point at
// non public hosts. The "http_url" tag already vets the scheme and overall
// shape. Here we only require the host to be a real FQDN, which
// excludes IP literals and single label hosts.
func validateCodeHostingURL(fl validator.FieldLevel) bool {
	parsed, err := url.Parse(fl.Field().String())
	if err != nil {
		return false
	}

	return hostValidator.Var(parsed.Hostname(), "fqdn") == nil
}

func GenerateErrorDetails(validationErrors []ValidationError) string {
	var errors []string

	for _, validationError := range validationErrors {
		switch validationError.Rule {
		case "required":
			errors = append(errors, validationError.Field+" is required")
		case "email":
			errors = append(errors, validationError.Field+" is not a valid email")
		case "min":
			errors = append(errors, validationError.Field+" does not meet its size limits (too short)")
		case "max":
			errors = append(errors, validationError.Field+" does not meet its size limits (too long)")
		case "gt":
			errors = append(errors, validationError.Field+" does not meet its size limits (too few items)")
		case "code_hosting_url":
			errors = append(errors, validationError.Field+" is not a valid public http(s) URL")
		default:
			errors = append(errors, validationError.Field+" is invalid")
		}
	}

	errorDetails := "invalid format: " + strings.Join(errors, ", ")

	return errorDetails
}
