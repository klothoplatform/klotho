package logging

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestSanitize(t *testing.T) {
	cases := []struct {
		name   string
		given  []zap.Field
		hasher func(any) string
		want   map[string]string
	}{
		{
			name: "non-sanitized field",
			given: []zap.Field{
				{
					Interface: "not a safe field",
				},
			},
			want: map[string]string{},
		},
		{
			name: "sanitized field unhashed",
			given: []zapcore.Field{
				{
					Interface: sanitizedString("safe to send"),
				},
			},
			want: map[string]string{
				"SanitizedField.sanitized": "safe to send",
			},
		},
		{
			name: "hashed field",
			given: []zapcore.Field{
				{
					Interface: hashingString("12345"),
				},
			},
			hasher: func(s any) string { return "this is a hash" },
			want: map[string]string{
				"HashingField.hashed": "this is a hash",
			},
		},
		{
			name: "hashed field no hasher",
			given: []zapcore.Field{
				{
					Interface: hashingString("12345"),
				},
			},
			want: map[string]string{
				"HashingField.hashed": "<redacted>",
			},
		},
		{
			name:  "no fields",
			given: []zap.Field{},
			want:  map[string]string{},
		},
		{
			name:  "nil fields",
			given: []zap.Field{},
			want:  map[string]string{},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			actual := SanitizeFields(tt.given, tt.hasher)
			assert.Equal(tt.want, actual)
		})
	}
}

type (
	sanitizedString string
	hashingString   string
)

func (s sanitizedString) Sanitize(hasher func(any) string) SanitizedField {
	return SanitizedField{
		Key:     "SanitizedField",
		Content: map[string]string{"sanitized": string(s)},
	}
}

func (s hashingString) Sanitize(hasher func(any) string) SanitizedField {
	return SanitizedField{
		Key:     "HashingField",
		Content: map[string]string{"hashed": hasher(string(s))},
	}
}
