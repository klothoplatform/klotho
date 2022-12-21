package logging

import (
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"testing"
)

func TestSanitize(t *testing.T) {
	cases := []struct {
		name   string
		given  []zap.Field
		hasher func(any) string
		want   string
	}{
		{
			name: "non-sanitized field",
			given: []zap.Field{
				{
					Interface: "not a safe field",
				},
			},
			want: "[]",
		},
		{
			name: "sanitized field unhashed",
			given: []zapcore.Field{
				{
					Interface: sanitizedString("safe to send"),
				},
			},
			want: `[{"key":"TestSanitizedField","content":"safe to send"}]`,
		},
		{
			name: "hashed field",
			given: []zapcore.Field{
				{
					Interface: hashingString("12345"),
				},
			},
			hasher: func(s any) string { return "this is a hash" },
			want:   `[{"key":"TestHashingField","content":"this is a hash"}]`,
		},
		{
			name: "hashed field no hasher",
			given: []zapcore.Field{
				{
					Interface: hashingString("12345"),
				},
			},
			want: `[{"key":"TestHashingField","content":"\u003credacted\u003e"}]`, // 0x3c is `<`, and 0x3e is `>`
		},
		{
			name:  "no fields",
			given: []zap.Field{},
			want:  "[]",
		},
		{
			name:  "nil fields",
			given: []zap.Field{},
			want:  "[]",
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
		Key:     "TestSanitizedField",
		Content: string(s),
	}
}

func (s hashingString) Sanitize(hasher func(any) string) SanitizedField {
	return SanitizedField{
		Key:     "TestHashingField",
		Content: hasher(string(s)),
	}
}
