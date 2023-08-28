package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/construct/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_PrivateDnsNamespaceCreate(t *testing.T) {
	eu := &types.ExecutionUnit{Name: "test"}
	eu2 := &types.ExecutionUnit{Name: "first"}
	initialRefs := construct.BaseConstructSetOf(eu)
	cases := []coretesting.CreateCase[PrivateDnsNamespaceCreateParams, *PrivateDnsNamespace]{
		{
			Name: "nil namespace",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:private_dns_namespace:my-app_pdns",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, namespace *PrivateDnsNamespace) {
				assert.Equal(namespace.Name, "my-app_pdns")
				assert.Equal(namespace.ConstructRefs, construct.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "existing namespace",
			Existing: &PrivateDnsNamespace{Name: "my-app_pdns", ConstructRefs: initialRefs},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:private_dns_namespace:my-app_pdns",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, namespace *PrivateDnsNamespace) {
				assert.Equal(namespace.Name, "my-app_pdns")
				initialRefs.Add(eu2)
				assert.Equal(namespace.ConstructRefs, initialRefs)
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = PrivateDnsNamespaceCreateParams{
				AppName: "my-app",
				Refs:    construct.BaseConstructSetOf(eu),
			}
			tt.Run(t)
		})
	}
}
