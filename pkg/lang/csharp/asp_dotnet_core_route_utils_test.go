package csharp

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSanitizeConventionalRoute(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "simple path",
			path:     "/hello",
			expected: "/hello",
		},
		{
			name:     "one path param",
			path:     "/hello/{world}",
			expected: "/hello/:world",
		}, {
			name:     "multiple path params",
			path:     "/hello/{other}/{world}",
			expected: "/hello/:other/:world",
		},
		{
			name:     "param with constraint",
			path:     "/hello/{world:int}",
			expected: "/hello/:world",
		},
		{
			name:     "optional param is converted to standard param",
			path:     "/hello/{world?}",
			expected: "/hello/:world",
		},
		{
			name:     "non-trailing default params are converted standard params",
			path:     "/api/my/{route=default}/with/defaults",
			expected: "/api/my/:route/with/defaults",
		},
		{
			name:     "multiple trailing default params are converted to proxy route",
			path:     "/api/my/{route=default}/{with:regex(pattern)=default}/{defaults:alpha=default}",
			expected: "/api/my/:rest*",
		},
		{
			name:     "trailing default params followed by a final optional param are converted to proxy route",
			path:     "/api/my/{route=default}/{with:regex((pattern))=default}/{defaults:alpha=default}/{optional?}",
			expected: "/api/my/:rest*",
		},
		{
			name:     "earlier path params are preserved when converting to wildcard",
			path:     "/hello/{other}/{**slug}",
			expected: "/hello/:other/:rest*",
		},
		{
			name:     "wildcards are converted to proxy routes 1",
			path:     "/api/{*}",
			expected: "/api/:rest*",
		},
		{
			name:     "wildcards are converted to proxy routes 2",
			path:     "/api/{**slug}",
			expected: "/api/:rest*",
		},
		{
			name:     "wildcards are converted to proxy routes 3",
			path:     "/api/{**slug}/trimmed",
			expected: "/api/:rest*",
		},
		{
			// Asterisk literals are not supported by AWS API Gateway
			name:     "asterisk literals ('*') are unmodified",
			path:     "/api/my*/strange/route**",
			expected: "/api/my*/strange/route**",
		},
		{
			name:     "complex segment 1",
			path:     "/{x}a{y}",
			expected: "/:complex1",
		},
		{
			name:     "complex segment 2",
			path:     "/{x}a{y}/a{x}b/",
			expected: "/:complex1/:complex2/",
		},
		{
			name:     "complex segment 3",
			path:     "{x}a{y}/",
			expected: ":complex1/",
		},
		{
			name:     "complex segment 4",
			path:     "x{x}",
			expected: ":complex1",
		},
		{
			name:     "complex segment 5",
			path:     "{x}.{y}",
			expected: ":complex1",
		},
		{
			name:     "regex constraints 1",
			path:     "/{p1:regex(pattern)}/{p2:regex(p{{a}}ttern)}",
			expected: "/:p1/:p2",
		},
		{
			name:     "regex constraints 2",
			path:     `/{p1:regex(patt\\\\\\)ern)}/{p2:regex(p\\)attern)}/`,
			expected: "/:p1/:p2/",
		},
		{
			name:     "regex constraints 3",
			path:     "{p1:regex(pattern)}",
			expected: ":p1",
		},
		{
			// This scenario verifies we don't get stuck in a loop processing bad input.
			// The original path will cause an HTTP-500 response from ASP.NET Core.
			name:     "invalid regex constraint results in bad path without causing an error",
			path:     "{p1:regex(pat}tern)}",
			expected: ":p1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, sanitizeConventionalPath(tt.path))
		})
	}
}

func Test_sanitizeAttributeBasedPath(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		area       string
		controller string
		action     string
		expected   string
	}{
		{
			name:     "simple path",
			path:     "/hello",
			expected: "/hello",
		},
		{
			// This scenario verifies we don't get stuck in a loop processing bad input.
			// The original path will cause an HTTP-500 response from ASP.NET Core.
			name:     "invalid regex constraint output bad path without causing an error",
			path:     "{p1:regex(pat}tern)}",
			expected: ":p1",
		},
		{
			name:     "one path param",
			path:     "/hello/{world}",
			expected: "/hello/:world",
		}, {
			name:     "multiple path params",
			path:     "/hello/{other}/{world}",
			expected: "/hello/:other/:world",
		},
		{
			name:     "param with constraint",
			path:     "/hello/{world:int}",
			expected: "/hello/:world",
		},
		{
			name:     "optional param is converted to standard param",
			path:     "/hello/{world?}",
			expected: "/hello/:world",
		},
		{
			name:     "non-trailing default params are converted standard params",
			path:     "/api/my/{route=default}/with/defaults",
			expected: "/api/my/:route/with/defaults",
		},
		{
			name:     "multiple trailing default params are converted to proxy route",
			path:     "/api/my/{route=default}/{with:regex(pattern)=default}/{defaults:alpha=default}",
			expected: "/api/my/:rest*",
		},
		{
			name:     "trailing single default param is converted to simple param",
			path:     "/api/my/{route}/{with}/{defaults=default}",
			expected: "/api/my/:route/:with/:defaults",
		},
		{
			name:     "trailing default params followed by a final optional param are converted to proxy route",
			path:     "/api/my/{route=default}/{with:regex((pattern))=default}/{defaults:alpha=default}/{optional?}",
			expected: "/api/my/:rest*",
		},
		{
			name:     "earlier path params are preserved when converting to wildcard",
			path:     "/hello/{other}/{**slug}",
			expected: "/hello/:other/:rest*",
		},
		{
			name:       "special tokens are replaced with their in-context values if the specified values match",
			path:       "/api/{area=MyArea}/{controller=MyController}/{action=MyAction}",
			expected:   "/api/MyArea/MyController/MyAction",
			area:       "MyArea",
			controller: "MyController",
			action:     "MyAction",
		},
		{
			name:       "special path params are ignored if their values do not match the current action's values",
			path:       "/api/{area=MyArea}/{controller=OtherController}/{action=OtherAction}",
			expected:   "/api/MyArea/:rest*",
			area:       "MyArea",
			controller: "MyController",
			action:     "MyAction",
		},
		{
			name:       "special tokens are replaced with their in-context values",
			path:       "/api/[area]/[CONTROLLER]/[Action]",
			expected:   "/api/MyArea/MyController/MyAction",
			area:       "MyArea",
			controller: "MyController",
			action:     "MyAction",
		},
		{
			name:     "wildcards are converted to proxy routes 1",
			path:     "/api/{*}",
			expected: "/api/:rest*",
		},
		{
			name:     "wildcards are converted to proxy routes 2",
			path:     "/api/{**slug}",
			expected: "/api/:rest*",
		},
		{
			name:     "wildcards are converted to proxy routes 3",
			path:     "/api/{**slug}/trimmed",
			expected: "/api/:rest*",
		},
		{
			name:     "complex segment 1",
			path:     "/{x}a{y}",
			expected: "/:complex1",
		},
		{
			name:     "complex segment 2",
			path:     "/{x}a{y}/{x}{y}/",
			expected: "/:complex1/:complex2/",
		},
		{
			name:     "complex segment 3",
			path:     "{x}a{y}/",
			expected: ":complex1/",
		},
		{
			name:     "complex segment 4",
			path:     "x{x}",
			expected: ":complex1",
		},
		{
			// Not a valid ASP.NET Core segment
			name:     "complex segment 5",
			path:     "{x}{y}",
			expected: ":complex1",
		},
		{
			name:     "regex constraint: treats escaped curly braces as literals",
			path:     "/{p1:regex(pattern)}/{p2:regex(p{{a}}ttern)}",
			expected: "/:p1/:p2",
		},
		{
			name:     "regex constraint: handles escaped parentheses",
			path:     `/{p1:regex(patt\\\\)=1}/{p2:regex(p\\)attern)}/`,
			expected: "/:p1/:p2/",
		},
		{
			name:     "regex constraint handled at start of path",
			path:     "{p1:regex(pattern)}",
			expected: ":p1",
		},
		{
			name:     "param with multiple constraints converted to simple param",
			path:     `{p1:regex(patt\\)er{{n}}):int:alpha=123}`,
			expected: ":p1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, sanitizeAttributeBasedPath(tt.path, tt.area, tt.controller, tt.action))
		})
	}
}
