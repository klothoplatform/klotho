package javascript

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/lang"
	"github.com/stretchr/testify/assert"
)

func TestFindAllCommentBlocks(t *testing.T) {
	cases := []lang.FindAllCommentBlocksTestCase{
		{
			Name: "single-line",
			Source: `
//@klotho::expose
x = y`,
			Want: []lang.FindAllCommentBlocksExpected{
				{
					Comment: "@klotho::expose",
					Node:    "x = y",
				},
			},
		},
		{
			Name: "multiple single-lines",
			Source: `
// @klotho::expose {
// foo = "bar"
// }
x = y`,
			Want: []lang.FindAllCommentBlocksExpected{
				{
					Comment: `@klotho::expose {
foo = "bar"
}`,
					Node: "x = y",
				},
			},
		},
		{
			Name: "simple multiline",
			Source: `
/*
* foo
*/
a = b`,
			Want: []lang.FindAllCommentBlocksExpected{
				{
					Comment: "\nfoo\n",
					Node:    "a = b",
				},
			},
		},
		{
			Name: "two multilines in a row",
			Source: `
/*
 * first
 */
/*
 second
 */
 a = b`,
			Want: []lang.FindAllCommentBlocksExpected{
				{
					Comment: "\nfirst\n\n\nsecond\n",
					Node:    "a = b",
				},
			},
		},
		{
			Name: "normal comment, then doc comment",
			Source: `
/*
 * comment starts with just one star
 */
/**
 * comment starts with two stars
 */
 a = b`,
			Want: []lang.FindAllCommentBlocksExpected{
				{
					Comment: "\ncomment starts with just one star\n\n\ncomment starts with two stars\n",
					Node:    "a = b",
				},
			},
		},
		{
			Name: "multi-line then single-line",
			Source: `
/*
 * first
 */
// second
a = b`,
			Want: []lang.FindAllCommentBlocksExpected{
				{
					Comment: "\nfirst\n\nsecond",
					Node:    "a = b",
				},
			},
		},
		{Name: "unclosed block comment",
			Source: `
/*
a = b`,
			Want: []lang.FindAllCommentBlocksExpected{},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			assert := assert.New(t)
			found, err := lang.FindAllCommentBlocksForTest(Language, tt.Source)
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.Want, found)
		})
	}
}
