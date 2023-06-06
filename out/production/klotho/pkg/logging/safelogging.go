package logging

import (
	"fmt"
	"go.uber.org/zap/zapcore"
)

type (
	Sanitizer interface {
		Sanitize(hasher func(any) string) SanitizedField
	}

	SanitizedField struct {
		Key     string
		Content map[string]string
	}
)

func SanitizeFields(fields []zapcore.Field, hasher func(any) string) map[string]string {
	if hasher == nil {
		hasher = func(_ any) string { return `<redacted>` }
	}
	result := make(map[string]string)
	for _, field := range fields {
		if safeField, isSafe := field.Interface.(Sanitizer); isSafe {
			sanitizedField := safeField.Sanitize(hasher)
			for k, v := range sanitizedField.Content {
				result[fmt.Sprintf("%s.%s", sanitizedField.Key, k)] = v
			}
		}
	}
	return result
}
