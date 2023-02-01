package lang

import (
	"strings"
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/stretchr/testify/assert"
)

func TestFindAllCommentBlocks(t *testing.T) {
	cases := []FindAllCommentBlocksTestCase{
		{"single-line",
			`// testing
const x = 123`,
			[]FindAllCommentBlocksExpected{
				{
					`testing`,
					`const x = 123`,
				},
			},
		},
		{"multi-line, no splits",
			`// first line
// second line
const x = 123`,
			[]FindAllCommentBlocksExpected{
				{
					"first line\nsecond line",
					"const x = 123",
				},
			},
		},
		{"two annotations",
			`// first
const x = 123

// second
const y = 456`,
			[]FindAllCommentBlocksExpected{
				{
					"first",
					"const x = 123",
				},
				{
					"second",
					"const y = 456",
				},
			},
		},
		{"comment is last node",
			`// only line in source`,
			[]FindAllCommentBlocksExpected{
				{
					"only line in source",
					"",
				},
			},
		},
		{"multi-line comment is last node",
			`// first line
// second line`,
			[]FindAllCommentBlocksExpected{
				{
					"first line\nsecond line",
					"",
				},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			assert := assert.New(t)
			found, err := FindAllCommentBlocksForTest(dummyJsLang, tt.Source)
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.Want, found)
		})
	}

	t.Run("sitter query has capture", func(t *testing.T) {
		assert := assert.New(t)

		lang := core.SourceLanguage{
			ID:               "capabilities_test_js",
			Sitter:           javascript.GetLanguage(),
			CapabilityFinder: NewCapabilityFinder("(comment) @c", func(in string) string { return in }),
		}
		found, err := FindAllCommentBlocksForTest(lang, "//comment\nx=y")
		if !assert.NoError(err) {
			return
		}
		want := []FindAllCommentBlocksExpected{
			{
				Comment: "//comment",
				Node:    "x=y",
			},
		}
		assert.Equal(want, found)
	})
}

func TestRegexpPreprocessor(t *testing.T) {
	pattern := `\s*REMOVE_ME`
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple",
			input: "1 REMOVE_ME 2 REMOVE_ME",
			want:  "1 2",
		},
		{
			name:  "no matches",
			input: "this is a test",
			want:  "this is a test",
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			preprocessor := RegexpRemovePreprocessor(pattern)
			foundString := preprocessor(tt.input)

			assert.Equal(tt.want, foundString)
		})
	}
}

func TestCompositePreprocessor(t *testing.T) {
	cases := []struct {
		name          string
		input         string
		preprocessors []CommentPreprocessor
		wantString    string
	}{
		{
			name:  "comment edits chain in order",
			input: "my FOO says hi",
			preprocessors: []CommentPreprocessor{
				func(comment string) string { return strings.Replace(comment, "FOO", "CAT", -1) },
				func(comment string) string { return strings.Replace(comment, "CAT", "DOG", -1) },
			},
			wantString: "my DOG says hi",
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			composite := CompositePreprocessor(tt.preprocessors...)
			foundString := composite(tt.input)

			assert.Equal(tt.wantString, foundString)
		})
	}
}

var dummyJsLang = core.SourceLanguage{
	ID:     "capabilities_test_js",
	Sitter: javascript.GetLanguage(),
	CapabilityFinder: NewCapabilityFinder("comment", func(comment string) string {
		return strings.Replace(comment, "// ", "", -1)
	}),
}
