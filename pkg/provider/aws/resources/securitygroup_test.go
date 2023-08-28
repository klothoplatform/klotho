package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/construct/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_SecurityGroupCreate(t *testing.T) {
	eu := &types.ExecutionUnit{Name: "test"}
	eu2 := &types.ExecutionUnit{Name: "first"}
	initialRefs := construct.BaseConstructSetOf(eu2)
	cases := []coretesting.CreateCase[SecurityGroupCreateParams, *SecurityGroup]{
		{
			Name: "nil igw",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:security_group:my-app",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, sg *SecurityGroup) {
				assert.Equal(sg.Name, "my-app")
				assert.Equal(sg.ConstructRefs, construct.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "existing igw",
			Existing: &SecurityGroup{Name: "my-app", ConstructRefs: initialRefs},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:security_group:my-app",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, sg *SecurityGroup) {
				assert.Equal(sg.Name, "my-app")
				assert.Equal(sg.ConstructRefs, construct.BaseConstructSetOf(eu, eu2))
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = SecurityGroupCreateParams{
				AppName: "my-app",
				Refs:    construct.BaseConstructSetOf(eu),
			}
			tt.Run(t)
		})
	}
}
