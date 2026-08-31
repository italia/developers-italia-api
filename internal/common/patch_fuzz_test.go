package common_test

import (
	"testing"

	"github.com/italia/open-catalog-api/internal/common"
)

func FuzzApplyPatchJSONPatch(f *testing.F) {
	f.Add([]byte(`[{"op": "replace", "path": "/name", "value": "bar"}]`))
	f.Add([]byte(`[{"op": "remove", "path": "/aliases/0"}]`))
	f.Add([]byte(`[{"op": "move", "from": "/name", "path": "/aliases/-"}]`))
	f.Add([]byte(`[]`))
	f.Add([]byte(`[{"op": "add", "path": "` + longPath(300) + `", "value": 1}]`))
	f.Add([]byte(`not json`))

	f.Fuzz(func(t *testing.T, body []byte) {
		entity := sampleEntity()

		_, err := common.ApplyPatch(&entity, common.ContentTypeJSONPatch, body)
		if err != nil && err.Code == 0 {
			t.Errorf("error with no HTTP status code for patch %q", body)
		}
	})
}

func FuzzApplyPatchMergePatch(f *testing.F) {
	f.Add([]byte(`{"name": "bar"}`))
	f.Add([]byte(`{"name": null}`))
	f.Add([]byte(`{"aliases": ["x"]}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`not json`))

	f.Fuzz(func(t *testing.T, body []byte) {
		entity := sampleEntity()

		_, err := common.ApplyPatch(&entity, "application/merge-patch+json", body)
		if err != nil && err.Code == 0 {
			t.Errorf("error with no HTTP status code for patch %q", body)
		}
	})
}

type fuzzEntity struct {
	Name    string   `json:"name"`
	Aliases []string `json:"aliases"`
	Active  bool     `json:"active"`
}

func sampleEntity() fuzzEntity {
	return fuzzEntity{Name: "foo", Aliases: []string{"a", "b"}, Active: true}
}

func longPath(length int) string {
	path := "/"
	for len(path) < length {
		path += "a"
	}

	return path
}
