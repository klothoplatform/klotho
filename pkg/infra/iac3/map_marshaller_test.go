package iac3

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTsMap_String(t *testing.T) {
	tests := []struct {
		name string
		m    TsMap
		want string
	}{
		{
			name: "empty map",
			m:    TsMap{},
			want: "{}",
		},
		{
			name: "map with one entry",
			m:    TsMap{"foo": "bar"},
			want: "{foo: bar}",
		},
		{
			name: "map with multiple entries",
			m:    TsMap{"foo": "bar", "baz": "qux"},
			want: "{foo: bar, baz: qux}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.m.String(), tt.want)
		})
	}
}

func TestTsList_String(t *testing.T) {
	tests := []struct {
		name string
		l    TsList
		want string
	}{
		{
			name: "empty list",
			l:    TsList{},
			want: "[]",
		},
		{
			name: "list with one entry",
			l:    TsList{"foo"},
			want: "[foo]",
		},
		{
			name: "list with multiple entries",
			l:    TsList{"foo", "bar"},
			want: "[foo, bar]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.l.String(), tt.want)
		})
	}
}
