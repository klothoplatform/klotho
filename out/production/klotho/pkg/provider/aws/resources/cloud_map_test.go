package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_PrivateDnsNamespaceCreate(t *testing.T) {
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}}
	initialRefs := core.AnnotationKeySetOf(core.AnnotationKey{ID: "first"})
	cases := []coretesting.CreateCase[PrivateDnsNamespaceCreateParams, *PrivateDnsNamespace]{
		{
			Name: "nil profile",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:private_dns_namespace:my-app",
					"aws:vpc:my_app",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:private_dns_namespace:my-app", Destination: "aws:vpc:my_app"},
				},
			},
			Check: func(assert *assert.Assertions, namespace *PrivateDnsNamespace) {
				assert.Equal(namespace.Name, "my-app")
				assert.NotNil(namespace.Vpc)
				assert.Equal(namespace.ConstructsRef, core.AnnotationKeySetOf(eu.AnnotationKey))
			},
		},
		{
			Name:     "existing lambda",
			Existing: &PrivateDnsNamespace{Name: "my-app", ConstructsRef: initialRefs},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:private_dns_namespace:my-app",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, namespace *PrivateDnsNamespace) {
				assert.Equal(namespace.Name, "my-app")
				initialRefs.Add(eu.AnnotationKey)
				assert.Equal(namespace.ConstructsRef, initialRefs)
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = PrivateDnsNamespaceCreateParams{
				AppName: "my-app",
				Refs:    core.AnnotationKeySetOf(eu.AnnotationKey),
			}
			tt.Run(t)
		})
	}
}
