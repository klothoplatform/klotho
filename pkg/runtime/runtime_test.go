package runtime

import (
	"strings"
	"testing"

	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/lang/dockerfile"
	"github.com/stretchr/testify/assert"
)

func Test_OverrideDockerfile(t *testing.T) {
	cases := []struct {
		name   string
		source string
		want   bool
	}{
		{
			name: "should not override",
			source: `
# @klotho::execution_unit { 
# id = "unit"
# }
FROM public.ecr.aws/lambda/nodejs:16				
			`,
			want: false,
		},
		{
			name: "should override due to id mismatch",
			source: `
# @klotho::execution_unit { 
# id = "not-the-unit"
# }
FROM public.ecr.aws/lambda/nodejs:16				
			`,
			want: true,
		},
		{
			name: "should not override due to no annotation",
			source: `
FROM public.ecr.aws/lambda/nodejs:16				
			`,
			want: true,
		},
		{
			name:   "should override due to no dockerfile",
			source: "",
			want:   true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			unit := types.ExecutionUnit{Name: "unit"}
			if tt.source != "" {
				f, _ := dockerfile.NewFile("Dockerfile", strings.NewReader(tt.source))
				unit.Add(f)
			}
			assert.Equal(tt.want, ShouldOverrideDockerfile(&unit))
		})
	}
}
