package lang

import (
	"regexp"
	"strings"
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/stretchr/testify/assert"
)

func TestFindAllCommentBlocks(t *testing.T) {
	cases := []FindAllCommentBlocksTestCase{
		{"multiline comment annotates its succeeding sibling node", `
		/* testing */
		const x = 123`,
			[]FindAllCommentBlocksExpected{
				{
					`testing `,
					`const x = 123`,
				},
			},
		},
		{"doc comment annotates its succeeding sibling node", `
		/** testing */
		const x = 123`,
			[]FindAllCommentBlocksExpected{
				{
					`testing `,
					`const x = 123`,
				},
			},
		},
		{"multiline comments are not merged with any surrounding comments", `
		// comment 1
		/* comment 2 */
		/* comment 3 */
		// comment 4
		const x = 123`,
			[]FindAllCommentBlocksExpected{
				{
					`comment 1`,
					``,
				},
				{
					`comment 2 `,
					``,
				},
				{
					`comment 3 `,
					``,
				},
				{
					`comment 4`,
					`const x = 123`,
				},
			},
		},
		{"single line comment block annotates its succeeding sibling node", `
		// testing
		const x = 123`,
			[]FindAllCommentBlocksExpected{
				{
					`testing`,
					`const x = 123`,
				},
			},
		},
		{"uninterrupted sequential line comments are combined into a single comment block", `
		// separate block
		
		// first line
		// second line
		// third line
		const x = 123`,
			[]FindAllCommentBlocksExpected{
				{
					"separate block",
					"",
				},
				{
					"first line\nsecond line\nthird line",
					"const x = 123",
				},
			},
		},
		{"each line comment block annotates its succeeding sibling node", `
		// first
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
		{"line comment as last node in file does not annotate a node",
			`// only line in source`,
			[]FindAllCommentBlocksExpected{
				{
					"only line in source",
					"",
				},
			},
		},
		{"line comment block as last node in file does not annotate a node", `
		// first line
		// second line`,
			[]FindAllCommentBlocksExpected{
				{
					"first line\nsecond line",
					"",
				},
			},
		},
		{"multiline comment as last node in file does not annotate a node", `
		/* 
		first line
		second line
		*/`,
			[]FindAllCommentBlocksExpected{
				{
					"first line\nsecond line\n",
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
			CapabilityFinder: NewCapabilityFinder("(comment) @c", func(in string) string { return in }, IsCLineCommentBlock),
		}
		found, err := FindAllCommentBlocksForTest(lang, "//comment\nx=y")
		if !assert.NoError(err) {
			return
		}
		want := []FindAllCommentBlocksExpected{
			{
				Comment:       "//comment",
				AnnotatedNode: "x=y",
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
	CapabilityFinder: NewCapabilityFinder("comment", CompositePreprocessor(
		RegexpRemovePreprocessor(`//\s*`),
		func(comment string) string {
			if !strings.HasPrefix(comment, "/*") {
				return comment
			}
			comment = comment[2 : len(comment)-2]
			comment = regexp.MustCompile(`(?m)^\s*[*]*[ \t]*`).ReplaceAllString(comment, "")
			return comment
		}),
		IsCLineCommentBlock),
}
