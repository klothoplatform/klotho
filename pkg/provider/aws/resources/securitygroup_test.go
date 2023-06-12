package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_SecurityGroupCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu)
	cases := []struct {
		name string
		sg   *SecurityGroup
		want coretesting.ResourcesExpectation
	}{
		{
			name: "nil sg",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:security_group:my_app:my-app",
					"aws:vpc:my_app",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:security_group:my_app:my-app", Destination: "aws:vpc:my_app"},
				},
			},
		},
		{
			name: "existing sg",
			sg:   &SecurityGroup{Name: "my-app", ConstructsRef: initialRefs, Vpc: &Vpc{Name: "my_app"}},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:security_group:my_app:my-app",
					"aws:vpc:my_app",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:security_group:my_app:my-app", Destination: "aws:vpc:my_app"},
				},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			if tt.sg != nil {
				dag.AddDependenciesReflect(tt.sg)
			}
			metadata := SecurityGroupCreateParams{
				AppName: "my-app",
				Refs:    core.BaseConstructSetOf(&core.ExecutionUnit{Name: "test"}),
			}
			sg := &SecurityGroup{}
			err := sg.Create(dag, metadata)

			if !assert.NoError(err) {
				return
			}
			tt.want.Assert(t, dag)

			graphSG := dag.GetResource(sg.Id())
			sg = graphSG.(*SecurityGroup)

			assert.Equal(sg.Name, "my-app")
			if tt.sg == nil {
				assert.Equal(sg.ConstructsRef, metadata.Refs)
			} else {
				expect := initialRefs.CloneWith(metadata.Refs)
				assert.Equal(sg.BaseConstructsRef(), expect)
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
