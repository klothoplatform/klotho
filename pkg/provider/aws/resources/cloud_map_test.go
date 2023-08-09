package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_PrivateDnsNamespaceCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test"}
	eu2 := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu)
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
				assert.Equal(namespace.ConstructRefs, core.BaseConstructSetOf(eu))
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
				Refs:    core.BaseConstructSetOf(eu),
			}
			tt.Run(t)
		})
	}
}
