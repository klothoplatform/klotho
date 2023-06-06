package dockerfile

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/lang"
	"github.com/stretchr/testify/assert"
)

func TestFindAllCommentBlocks(t *testing.T) {
	cases := []lang.FindAllCommentBlocksTestCase{
		{Name: "simple single line",
			Source: `
# one
FROM public.ecr.aws/lambda/nodejs:16`,
			Want: []lang.FindAllCommentBlocksExpected{
				{
					Comment:       "one",
					AnnotatedNode: "FROM public.ecr.aws/lambda/nodejs:16",
				},
			},
		},
		{Name: "indented comment single line",
			Source: `
FROM public.ecr.aws/lambda/nodejs:16
  # @klotho::execution_unit
COPY . ./`,
			Want: []lang.FindAllCommentBlocksExpected{
				{
					Comment:       "@klotho::execution_unit",
					AnnotatedNode: "COPY . ./",
				},
			},
		},
		{Name: "simple block",
			Source: `
# one
# two
FROM public.ecr.aws/lambda/nodejs:16`,
			Want: []lang.FindAllCommentBlocksExpected{
				{
					Comment:       "one\ntwo",
					AnnotatedNode: "FROM public.ecr.aws/lambda/nodejs:16",
				},
			},
		},
		{Name: "indented comment block",
			Source: `
FROM public.ecr.aws/lambda/nodejs:16
  # @klotho::execution_unit {
  #    id = "dockerfile-unit"
  # }
COPY . ./`,
			Want: []lang.FindAllCommentBlocksExpected{
				{
					Comment:       "@klotho::execution_unit {\nid = \"dockerfile-unit\"\n}",
					AnnotatedNode: "COPY . ./",
				},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			assert := assert.New(t)
			found, err := lang.FindAllCommentBlocksForTest(language, tt.Source)
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.Want, found)
		})
	}
}
