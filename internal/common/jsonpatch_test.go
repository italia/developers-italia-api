package common

import (
	"strings"
	"testing"

	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustDecodePatch(t *testing.T, raw string) jsonpatch.Patch {
	t.Helper()

	patch, err := jsonpatch.DecodePatch([]byte(raw))
	require.NoError(t, err)

	return patch
}

func TestValidateJSONPatch(t *testing.T) {
	t.Run("valid single op", func(t *testing.T) {
		patch := mustDecodePatch(t, `[{"op":"replace","path":"/active","value":true}]`)

		assert.NoError(t, ValidateJSONPatch(patch))
	})

	t.Run("empty patch returns error", func(t *testing.T) {
		patch := mustDecodePatch(t, `[]`)

		assert.ErrorIs(t, ValidateJSONPatch(patch), errJSONPatchEmpty)
	})

	t.Run("patch with 101 ops returns error", func(t *testing.T) {
		var ops []string
		for range 101 {
			ops = append(ops, `{"op":"replace","path":"/active","value":true}`)
		}

		patch := mustDecodePatch(t, "["+strings.Join(ops, ",")+"]")

		assert.ErrorIs(t, ValidateJSONPatch(patch), errJSONPatchTooMany)
	})

	t.Run("path longer than 255 chars returns error", func(t *testing.T) {
		longPath := "/" + strings.Repeat("x", 255)
		raw := `[{"op":"replace","path":"` + longPath + `","value":true}]`
		patch := mustDecodePatch(t, raw)

		assert.ErrorIs(t, ValidateJSONPatch(patch), errJSONPatchPathLen)
	})

	t.Run("path exactly 255 chars is valid", func(t *testing.T) {
		okPath := "/" + strings.Repeat("x", 254)
		raw := `[{"op":"replace","path":"` + okPath + `","value":true}]`
		patch := mustDecodePatch(t, raw)

		assert.NoError(t, ValidateJSONPatch(patch))
	})
}
