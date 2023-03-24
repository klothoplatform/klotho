package iac2

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSnakeToLower(t *testing.T) {
	cases := map[string]string{
		"FooBar":  "foo_bar",
		"fooBar":  "foo_bar",
		"foo_bar": "foo_bar",
		"foo_Bar": "foo_bar",
	}
	for orig, want := range cases {
		t.Run(fmt.Sprintf("[%s]>[%s]", orig, want), func(t *testing.T) {
			assert := assert.New(t)
			assert.Equal(want, camelToSnake(orig))
		})
	}
}

func TestToUpperCamel(t *testing.T) {
	cases := map[string]string{
		"foo_bar":   "FooBar",
		"FooBar":    "FooBar",
		"fooBar":    "FooBar",
		"foo_Bar":   "FooBar",
		"foo-Bar":   "FooBar",
		"foo-_:Bar": "FooBar",
		"_fooBar":   "FooBar",
		"fooBar_":   "FooBar",
	}
	for orig, want := range cases {
		t.Run(fmt.Sprintf("[%s]>[%s]", orig, want), func(t *testing.T) {
			assert := assert.New(t)
			assert.Equal(want, toUpperCamel(orig))
		})
	}

}

func TestGetStructValues(t *testing.T) {
	assert := assert.New(t)
	s := MyStruct{
		MyInt:        123,
		MyStr:        "hello world",
		myPrivateInt: 456,
	}

	assert.Equal(
		map[string]any{
			"MyInt": 123,
			"MyStr": "hello world",
		},
		getStructValues(s),
	)
}

type MyStruct struct {
	MyInt        int
	MyStr        string
	myPrivateInt int
}
