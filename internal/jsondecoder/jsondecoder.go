package jsondecoder

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
)

var (
	ErrExtraDataAfterDecoding = errors.New("extra data after decoding")
	ErrUnknownField           = errors.New("unknown field in JSON input")
)

// UnmarshalDisallowUnknownFieldsUnmarshal parses the JSON-encoded data
// and stores the result in the value pointed to by v like json.Unmarshal,
// but with DisallowUnknownFields() set by default for extra security.
func UnmarshalDisallowUnknownFields(data []byte, v interface{}) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()

	if err := dec.Decode(v); err != nil {
		// Ugly, but the encoding/json uses a dynamic error here
		if strings.HasPrefix(err.Error(), "json: unknown field ") {
			return ErrUnknownField
		}

		// we want to provide an alternative implementation, with the
		// unwrapped errors
		//nolint:wrapcheck
		return err
	}

	// Check if there's any data left in the decoder's buffer.
	// This ensures that there's no extra JSON after the main object
	// otherwise something like '{"foo": 1}{"bar": 2}' or even '{}garbage'
	// will not error out.
	if dec.More() {
		return ErrExtraDataAfterDecoding
	}

	return nil
}
