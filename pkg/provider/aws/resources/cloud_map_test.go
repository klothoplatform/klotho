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
					"aws:private_dns_namespace:my-app",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, namespace *PrivateDnsNamespace) {
				assert.Equal(namespace.Name, "my-app")
				assert.Equal(namespace.ConstructsRef, core.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "existing namespace",
			Existing: &PrivateDnsNamespace{Name: "my-app", ConstructsRef: initialRefs},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:private_dns_namespace:my-app",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, namespace *PrivateDnsNamespace) {
				assert.Equal(namespace.Name, "my-app")
				initialRefs.Add(eu2)
				assert.Equal(namespace.ConstructsRef, initialRefs)
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

func Test_PrivateDnsNamespaceMakeOperational(t *testing.T) {
	cases := []coretesting.MakeOperationalCase[*PrivateDnsNamespace]{
		{
			Name:     "only PrivateDnsNamespace",
			Resource: &PrivateDnsNamespace{Name: "ns"},
			AppName:  "my-app",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:private_dns_namespace:ns",
					"aws:vpc:my_app",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:private_dns_namespace:ns", Destination: "aws:vpc:my_app"},
				},
			},
			Check: func(assert *assert.Assertions, ns *PrivateDnsNamespace) {
				assert.NotNil(ns.Vpc)
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Run(t)
		})
	}
}
