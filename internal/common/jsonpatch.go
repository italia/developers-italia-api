package common

import (
	"errors"
	"fmt"

	jsonpatch "github.com/evanphx/json-patch/v5"
)

const (
	maxJSONPatchOps     = 100
	maxJSONPatchPathLen = 255
)

var (
	errJSONPatchEmpty   = errors.New("patch must contain at least one operation")
	errJSONPatchTooMany = fmt.Errorf("patch must not exceed %d operations", maxJSONPatchOps)
	errJSONPatchPathLen = fmt.Errorf("path and from must not exceed %d characters", maxJSONPatchPathLen)
)

// ValidateJSONPatch checks that a decoded patch satisfies the OAS constraints:
// 1 to 100 operations, path and from at most 255 characters each.
func ValidateJSONPatch(patch jsonpatch.Patch) error {
	if len(patch) < 1 {
		return errJSONPatchEmpty
	}

	if len(patch) > maxJSONPatchOps {
		return errJSONPatchTooMany
	}

	for _, op := range patch {
		if path, err := op.Path(); err == nil && len(path) > maxJSONPatchPathLen {
			return errJSONPatchPathLen
		}

		if from, err := op.From(); err == nil && len(from) > maxJSONPatchPathLen {
			return errJSONPatchPathLen
		}
	}

	return nil
}
