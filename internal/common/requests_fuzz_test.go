package common_test

import (
	"testing"

	"github.com/italia/open-catalog-api/internal/common"
	"github.com/italia/open-catalog-api/internal/jsondecoder"
)

// FuzzValidatePostRequests mirrors the POST body path: Fiber's BodyParser
// is wired to jsondecoder.UnmarshalDisallowUnknownFields, then the decoded
// request goes through ValidateStruct and GenerateErrorDetails.
func FuzzValidatePostRequests(f *testing.F) {
	f.Add([]byte(`{"name": "Comune di Bugliano"}`))
	f.Add([]byte(`{"url": "https://example.org/repo.git", "publiccodeYml": "publiccodeYmlVersion: \"0\""}`))
	f.Add([]byte(`{"codeHosting": [{"url": "https://example.org"}], "description": "d"}`))
	f.Add([]byte(`{"codeHosting": [{"url": "http://127.0.0.1"}], "description": "d"}`))
	f.Add([]byte(`{"email": "not-an-email", "description": ""}`))
	f.Add([]byte(`{"url": "https://example.org/hook", "secret": "0123456789abcdef"}`))
	f.Add([]byte(`{"message": "hello"}`))
	f.Add([]byte(`{"sources": [{"url": ":\\not a url", "args": [""]}]}`))

	f.Fuzz(func(t *testing.T, body []byte) {
		for _, request := range []any{
			&common.CatalogPost{},
			&common.PublisherPost{},
			&common.SoftwarePost{},
			&common.Webhook{},
			&common.Log{},
		} {
			if err := jsondecoder.UnmarshalDisallowUnknownFields(body, request); err != nil {
				continue
			}

			if validationErrors := common.ValidateStruct(request); validationErrors != nil {
				_ = common.GenerateErrorDetails(validationErrors)
			}
		}
	})
}
