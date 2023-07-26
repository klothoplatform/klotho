package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_LambdaCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test"}
	eu2 := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu2)
	cases := []coretesting.CreateCase[LambdaCreateParams, *LambdaFunction]{
		{
			Name: "nil function",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:lambda_function:my-app-function",
					"aws:log_group:my-app-function",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:lambda_function:my-app-function", Destination: "aws:log_group:my-app-function"},
				},
			},
			Check: func(assert *assert.Assertions, l *LambdaFunction) {
				assert.Equal(l.Name, "my-app-function")
				assert.Equal(l.ConstructRefs, core.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "existing function",
			Existing: &LambdaFunction{Name: "my-app-function", ConstructRefs: initialRefs},
			WantErr:  true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = LambdaCreateParams{
				Refs:    core.BaseConstructSetOf(eu),
				AppName: "my-app",
				Name:    "function",
			}
			tt.Run(t)
		})
	}
}
