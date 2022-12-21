package python

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/lang"
	"github.com/stretchr/testify/assert"
)

func TestFindAllCommentBlocks(t *testing.T) {
	cases := []lang.FindAllCommentBlocksTestCase{
		{Name: "simple",
			Source: `
# one
# two
x = y`,
			Want: []lang.FindAllCommentBlocksExpected{
				{
					Comment: "one\ntwo",
					Node:    "x = y",
				},
			},
		},
		{Name: "indented comment",
			Source: `
def foo():
    # @klotho::fizz
	return None`,
			Want: []lang.FindAllCommentBlocksExpected{
				{
					Comment: "@klotho::fizz",
					Node:    "return None",
				},
			},
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
