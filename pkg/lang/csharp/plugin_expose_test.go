package csharp

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSanitizeConventionalRoute(t *testing.T) {
	tests := []struct {
		name     string
		route    string
		expected string
	}{
		{
			name:     "simple route",
			route:    "/hello",
			expected: "/hello",
		},
		{
			name:     "one path param",
			route:    "/hello/{world}",
			expected: "/hello/:world",
		}, {
			name:     "multiple path params",
			route:    "/hello/{other}/{world}",
			expected: "/hello/:other/:world",
		},
		{
			name:     "param with constraint",
			route:    "/hello/{world:int}",
			expected: "/hello/:world",
		},
		{
			name:     "optional param is converted to wildcard",
			route:    "/hello/{world?}",
			expected: "/hello/:rest*",
		},
		{
			name:     "default param is converted to wildcard",
			route:    "/hello/other/{world=default}",
			expected: "/hello/other/:rest*",
		},
		{
			name:     "earlier path params are preserved when converting to wildcard",
			route:    "/hello/{other}/{world=default}",
			expected: "/hello/:other/:rest*",
		},
		{
			name:     "route is converted to longest possible wildcard",
			route:    "/hello/far/away/{other?}/{world=default}",
			expected: "/hello/far/away/:rest*",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, sanitizeConventionalPath(tt.route))
		})
	}
}
