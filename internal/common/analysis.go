package common

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	jsonpatch "github.com/evanphx/json-patch/v5"
)

var (
	errAnalysisNotObject    = errors.New("must be a JSON object")
	errAnalysisMissingV     = errors.New("missing required field 'v'")
	errAnalysisInvalidV     = errors.New("'v' must be an integer")
	errAnalysisUnexpectedDB = errors.New("unexpected database value type for analysis")
)

// AnalysisData is a map of { namespace: arbitrary JSON object }
// External software components (scanners, enrichers, security checkers, etc.) write under
// their own key.
// Every namespace must have "v" (schema version, int).
type AnalysisData map[string]json.RawMessage //nolint:recvcheck

func (a AnalysisData) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil //nolint:nilnil
	}

	data, err := json.Marshal(a)
	if err != nil {
		return nil, fmt.Errorf("can't marshal AnalysisData: %w", err)
	}

	return string(data), nil
}

func (a *AnalysisData) Scan(value any) error {
	if value == nil {
		*a = nil

		return nil
	}

	var data []byte

	switch v := value.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		return errAnalysisUnexpectedDB
	}

	//nolint:wrapcheck
	return json.Unmarshal(data, a)
}

// namespaceMeta holds the fields we validate, the rest of the namespace is opaque.
type namespaceMeta struct {
	V *json.RawMessage `json:"v"`
}

// WithTimestamps checks that each namespace is a JSON object with an integer "v",
// then injects "t" (current time, RFC 3339).
func WithTimestamps(analysis AnalysisData, now time.Time) (AnalysisData, error) {
	if analysis == nil {
		return nil, nil //nolint:nilnil
	}

	tPatch, _ := json.Marshal(struct { //nolint:errchkjson
		T string `json:"t"`
	}{T: now.UTC().Format(time.RFC3339)})

	result := make(AnalysisData, len(analysis))

	for namespace, raw := range analysis {
		var meta namespaceMeta

		if err := json.Unmarshal(raw, &meta); err != nil {
			return nil, fmt.Errorf("analysis.%s: %w", namespace, errAnalysisNotObject)
		}

		if meta.V == nil {
			return nil, fmt.Errorf("analysis.%s: %w", namespace, errAnalysisMissingV)
		}

		var version int
		if err := json.Unmarshal(*meta.V, &version); err != nil {
			return nil, fmt.Errorf("analysis.%s: %w", namespace, errAnalysisInvalidV)
		}

		result[namespace], _ = jsonpatch.MergePatch(raw, tPatch)
	}

	return result, nil
}
