package yaml

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
resource: test`,
			Want: []lang.FindAllCommentBlocksExpected{
				{
					Comment:       "one",
					AnnotatedNode: "resource: test",
				},
			},
		},
		{Name: "indented comment single line",
			Source: `
maintainers:
- email: kubernetes@nginx.com
  # @klotho::execution_unit
  name: nginxinc
`,
			Want: []lang.FindAllCommentBlocksExpected{
				{
					Comment:       "@klotho::execution_unit",
					AnnotatedNode: "name: nginxinc",
				},
			},
		},
		{Name: "simple block",
			Source: `
# one
# two
resource: test`,
			Want: []lang.FindAllCommentBlocksExpected{
				{
					Comment:       "one\ntwo",
					AnnotatedNode: "resource: test",
				},
			},
		},
		{Name: "indented comment block",
			Source: `
maintainers:
  # @klotho::execution_unit {
  #    id = "nginx-helm"
  # }
  email: test`,
			Want: []lang.FindAllCommentBlocksExpected{
				{
					Comment:       "@klotho::execution_unit {\nid = \"nginx-helm\"\n}",
					AnnotatedNode: "email: test",
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
