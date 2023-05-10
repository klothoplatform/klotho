package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_SecurityGroupCreate(t *testing.T) {
	initialRefs := []core.AnnotationKey{{ID: "first"}}
	cases := []struct {
		name string
		sg   *SecurityGroup
		want coretesting.ResourcesExpectation
	}{
		{
			name: "nil role",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:security_group:my-app",
					"aws:vpc:my_app",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:security_group:my-app", Destination: "aws:vpc:my_app"},
				},
			},
		},
		{
			name: "existing role",
			sg:   &SecurityGroup{Name: "my-app", ConstructsRef: initialRefs},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:security_group:my-app",
				},
				Deps: []coretesting.StringDep{},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			if tt.sg != nil {
				dag.AddResource(tt.sg)
			}
			metadata := SecurityGroupCreateParams{
				AppName: "my-app",
				Refs:    []core.AnnotationKey{{ID: "test", Capability: annotation.ExecutionUnitCapability}},
			}
			sg := &SecurityGroup{}
			err := sg.Create(dag, metadata)

			if !assert.NoError(err) {
				return
			}
			tt.want.Assert(t, dag)

			assert.Equal(sg.Name, "my-app")
			if tt.sg == nil {
				assert.Len(sg.IngressRules, 2)
				assert.Len(sg.EgressRules, 1)
				assert.Equal(sg.ConstructsRef, metadata.Refs)
			} else {
				sg := dag.GetResourceByVertexId(sg.Id().String())
				assert.Equal(sg, tt.sg)
				assert.ElementsMatch(sg.KlothoConstructRef(), append(initialRefs, core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}))
			}
		})
	}
}

func Test_GetSecurityGroup(t *testing.T) {
	vpc := NewVpc("test")
	type result struct {
		ingressRules []SecurityGroupRule
		egressRules  []SecurityGroupRule
	}
	cases := []struct {
		name       string
		existingSG *SecurityGroup
		want       result
	}{
		{
			name: "new SG is created",
			want: result{
				ingressRules: []SecurityGroupRule{
					{
						Description: "Allow ingress traffic from ip addresses within the vpc",
						CidrBlocks: []core.IaCValue{
							{Resource: vpc, Property: CIDR_BLOCK_IAC_VALUE},
						},
						FromPort: 0,
						Protocol: "-1",
						ToPort:   0,
					},
					{
						Description: "Allow ingress traffic from within the same security group",
						FromPort:    0,
						Protocol:    "-1",
						ToPort:      0,
						Self:        true,
					},
				},
				egressRules: []SecurityGroupRule{
					{
						Description: "Allows all outbound IPv4 traffic.",
						FromPort:    0,
						Protocol:    "-1",
						ToPort:      0,
						CidrBlocks: []core.IaCValue{
							{Property: "0.0.0.0/0"},
						},
					},
				},
			},
		},
		{
			name:       "existing sg",
			existingSG: &SecurityGroup{},
			want: result{
				ingressRules: []SecurityGroupRule{},
				egressRules:  []SecurityGroupRule{},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			cfg := &config.Application{}

			dag.AddResource(vpc)
			if tt.existingSG != nil {
				dag.AddResource(tt.existingSG)
			}

			result := GetSecurityGroup(cfg, dag)
			assert.ElementsMatch(result.IngressRules, tt.want.ingressRules)
			assert.ElementsMatch(result.EgressRules, tt.want.egressRules)
		})
	}
}
