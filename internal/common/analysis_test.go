package common

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithTimestamps(t *testing.T) {
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	t.Run("nil returns nil", func(t *testing.T) {
		result, err := WithTimestamps(nil, now)

		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("injects t and preserves other fields", func(t *testing.T) {
		input := AnalysisData{
			"scanner": json.RawMessage(`{"v": 1, "score": 90}`),
		}

		result, err := WithTimestamps(input, now)
		require.NoError(t, err)

		var ns map[string]interface{}
		require.NoError(t, json.Unmarshal(result["scanner"], &ns))

		assert.Equal(t, float64(1), ns["v"])
		assert.Equal(t, float64(90), ns["score"])
		assert.Equal(t, "2024-01-15T10:30:00Z", ns["t"])
	})

	t.Run("injects t into all namespaces", func(t *testing.T) {
		input := AnalysisData{
			"scanner":  json.RawMessage(`{"v": 1}`),
			"enricher": json.RawMessage(`{"v": 2}`),
		}

		result, err := WithTimestamps(input, now)
		require.NoError(t, err)

		for _, ns := range []string{"scanner", "enricher"} {
			var m map[string]interface{}
			require.NoError(t, json.Unmarshal(result[ns], &m))
			assert.Equal(t, "2024-01-15T10:30:00Z", m["t"])
		}
	})

	t.Run("t overwrites existing t", func(t *testing.T) {
		input := AnalysisData{
			"scanner": json.RawMessage(`{"v": 1, "t": "2000-01-01T00:00:00Z"}`),
		}

		result, err := WithTimestamps(input, now)
		require.NoError(t, err)

		var ns map[string]interface{}
		require.NoError(t, json.Unmarshal(result["scanner"], &ns))

		assert.Equal(t, "2024-01-15T10:30:00Z", ns["t"])
	})

	t.Run("missing v returns error", func(t *testing.T) {
		input := AnalysisData{
			"scanner": json.RawMessage(`{"score": 90}`),
		}

		_, err := WithTimestamps(input, now)

		assert.ErrorIs(t, err, errAnalysisMissingV)
	})

	t.Run("v not integer returns error", func(t *testing.T) {
		input := AnalysisData{
			"scanner": json.RawMessage(`{"v": "one"}`),
		}

		_, err := WithTimestamps(input, now)

		assert.ErrorIs(t, err, errAnalysisInvalidV)
	})

	t.Run("namespace not an object returns error", func(t *testing.T) {
		input := AnalysisData{
			"scanner": json.RawMessage(`[1, 2, 3]`),
		}

		_, err := WithTimestamps(input, now)

		assert.ErrorIs(t, err, errAnalysisNotObject)
	})
}
