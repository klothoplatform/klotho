package cli

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"gopkg.in/yaml.v3"
	"os"
	"testing"
)

func TestSetOptions(t *testing.T) {
	cases := []struct {
		name         string
		set          string
		to           string
		originalYaml string
		expectYaml   string
		expectErr    bool
		expectWarns  []string
	}{
		{
			name: "valid option",
			// blank vs non-blank isn't too interesting, because yaml_util handles that. See its tests for details.
			originalYaml: "",
			set:          "update.stream",
			to:           "my-stream",
			expectYaml:   "update:\n    stream: my-stream\n",
		},
		{
			name:      "invalid option",
			set:       "update",
			to:        "this can't be a scalar",
			expectErr: true,
		},
		{
			name:        "unknown option",
			set:         "foo.bar",
			to:          "world",
			expectYaml:  "foo:\n    bar: world\n",
			expectWarns: []string{`Unrecognized option "foo.bar". We'll still set it, but it may not have any effect.`},
		},
		{
			name:         "yaml starts with unknown option",
			set:          "update.stream",
			to:           "my-stream",
			originalYaml: "extra: option",
			expectYaml:   "extra: option\nupdate:\n    stream: my-stream\n",
			expectWarns: []string{
				`Existing options contain extra parameters:`,
				`▸ line 1: field extra not found in type cli.Options`,
			},
		},
		{
			name:         "yaml starts invalid",
			set:          "update.stream",
			to:           "my-stream",
			originalYaml: "update: this can't be a scalar",
			expectErr:    true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			options := map[string]string{
				tt.set: tt.to,
			}

			observedZapCore, observedLogs := observer.New(zap.WarnLevel)
			observedLogger := zap.New(observedZapCore)
			defer zap.ReplaceGlobals(observedLogger)()

			result, err := setOptions([]byte(tt.originalYaml), options)

			var seenLogs []string
			for _, entry := range observedLogs.All() {
				seenLogs = append(seenLogs, entry.Message)
			}
			assert.Equal(tt.expectWarns, seenLogs)
			assert.Equal(tt.expectErr, err != nil)
			if err != nil {
				return
			}
			assert.Equal(tt.expectYaml, string(result))
		})
	}
}

func TestCliSerialization(t *testing.T) {
	_, err := os.ReadFile("adslkfjadsf")
	errors.Is(err, os.ErrNotExist)
	println("%s", err)
	cases := []struct {
		name   string
		given  Options
		expect string
	}{
		{
			name: "normal",
			given: Options{
				Update: UpdateOptions{
					Stream: "my-stream",
				},
			},
			expect: "update:\n    stream: my-stream\n",
		},
		{
			name: "no update stream",
			given: Options{
				Update: UpdateOptions{},
			},
			expect: "{}\n",
		},
		{
			name:   "no update options",
			given:  Options{},
			expect: "{}\n",
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			actual, err := yaml.Marshal(tt.given)
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.expect, string(actual))
		})
	}

}
func TestCliDeserialization(t *testing.T) {
	cases := []struct {
		name      string
		given     string
		expect    Options
		expectErr bool
	}{
		{
			name:  "normal",
			given: "update:\n    stream: my-stream\n",
			expect: Options{
				Update: UpdateOptions{
					Stream: "my-stream",
				},
			},
		},
		{
			name:  "minimal",
			given: "{}",
			expect: Options{
				Update: UpdateOptions{},
			},
		},
		{
			name:  "empty",
			given: "",
			expect: Options{
				Update: UpdateOptions{},
			},
		},
		{
			name:  "extra values are ignored",
			given: "hello: world",
			expect: Options{
				Update: UpdateOptions{},
			},
		},
		{
			name:      "invalid values", // ie an "update" key that doesn't point to a struct
			given:     "update: yes",
			expectErr: true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			actual := Options{}
			err := yaml.Unmarshal([]byte(tt.given), &actual)

			assert.Equal(tt.expectErr, err != nil, "actual error: %s", err)
			assert.Equal(tt.expect, actual)
		})
	}
}
