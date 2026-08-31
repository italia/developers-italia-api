package jsondecoder_test

import (
	"encoding/json"
	"testing"

	"github.com/italia/open-catalog-api/internal/jsondecoder"
)

func FuzzUnmarshalDisallowUnknownFields(f *testing.F) {
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"name": "foo"}`))
	f.Add([]byte(`{"name": "foo"}{"name": "bar"}`))
	f.Add([]byte(`{"unknown": true}`))
	f.Add([]byte(`{}garbage`))
	f.Add([]byte(`null`))
	f.Add([]byte(`[1, 2, 3]`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var target struct {
			Name string `json:"name"`
		}

		err := jsondecoder.UnmarshalDisallowUnknownFields(data, &target)
		if err == nil && !json.Valid(data) {
			t.Errorf("accepted invalid JSON: %q", data)
		}
	})
}
