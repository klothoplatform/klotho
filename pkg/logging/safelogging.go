package logging

import (
	"encoding/json"
	"go.uber.org/zap/zapcore"
)

type (
	Sanitizer interface {
		Sanitize(hasher func(any) string) SanitizedField
	}

	SanitizedField struct {
		Key     string `json:"key"`
		Content any    `json:"content,omitempty"`
	}
)

// JsonMarshalSafely tries to marshal the value, but returns `[]byte("<error>")` if it can't (instead of an error).
func JsonMarshalSafely(v any) []byte {
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return []byte("<error>")
	}
	return jsonBytes
}

func SanitizeFields(fields []zapcore.Field, hasher func(any) string) string {
	if hasher == nil {
		hasher = func(_ any) string { return `<redacted>` }
	}
	safeLogs := make([]SanitizedField, 0, len(fields))
	for _, field := range fields {
		if safeField, isSafe := field.Interface.(Sanitizer); isSafe {
			safeLogs = append(safeLogs, safeField.Sanitize(hasher))
		}
	}
	return string(JsonMarshalSafely(safeLogs))
}
