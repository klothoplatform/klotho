package python

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPipClone(t *testing.T) {
	orig := &RequirementsTxt{
		path:     "my-reqs.txt",
		contents: []byte("file contents"),
		extras:   []string{"one"},
	}
	cloneFile := orig.Clone()
	clonePip, ok := cloneFile.(*RequirementsTxt)
	if !assert.True(t, ok, "clone failed") {
		return
	}

	t.Run("contents array cloned", func(t *testing.T) {
		assert := assert.New(t)
		clonePip.contents[0] = 'P'
		assert.Equal([]byte("file contents"), orig.contents)     // Sanity check that this test isn't buggy
		assert.Equal([]byte("Pile contents"), clonePip.contents) // Clone should be unchanged by orig's changes
	})
	t.Run("extras array cloned", func(t *testing.T) {
		assert := assert.New(t)
		// Check that the "extras" slice got copied, not shared
		assert.Equal([]string{"one"}, clonePip.extras)
		clonePip.extras[0] = "Two"
		assert.Equal([]string{"one"}, orig.extras)
		assert.Equal([]string{"Two"}, clonePip.extras)
	})
	t.Run("path cloned", func(t *testing.T) {
		assert := assert.New(t)
		assert.Equal("my-reqs.txt", clonePip.path)
	})
}

func TestNewRequirementsTxtPath(t *testing.T) {
	assert := assert.New(t)

	pip, err := NewRequirementsTxt("requirements.txt", strings.NewReader("original contents"))
	if !assert.NoError(err) {
		return
	}
	// Really just makes sure that NewPipeFile created it correctly
	assert.Equal("requirements.txt", pip.Path())
}

func TestPipWrite(t *testing.T) {
	tests := []struct {
		name             string
		originalContents string
		extras           []string
		want             string
	}{

		{
			name:             "no extra lines",
			originalContents: "hello world",
			want:             "hello world",
		},
		{
			name:             "some extras",
			originalContents: "original contents",
			extras:           []string{"one", "two"},
			want: `original contents

# Added by Klotho:
one
two
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			pip, err := NewRequirementsTxt("requirements.txt", strings.NewReader(tt.originalContents))
			if !assert.NoError(err) {
				return
			}

			for _, extra := range tt.extras {
				pip.AddLine(extra)
			}

			var sb strings.Builder
			_, err = pip.WriteTo(&sb)
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.want, sb.String())
		})
	}
	t.Run("nil contents and extras", func(t *testing.T) {
		assert := assert.New(t)
		// Not doing this as a case from above, since handling the nil handling would complicate the "normal" handling
		pip := &RequirementsTxt{}
		var sb strings.Builder
		_, err := pip.WriteTo(&sb)
		if !assert.NoError(err) {
			return
		}
		assert.Equal("", sb.String())

	})
}
