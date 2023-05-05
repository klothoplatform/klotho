package resources

import (
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
)

type (
	SecurityGroup struct {
		Name          string
		Vpc           *Vpc
		ConstructsRef []core.AnnotationKey
		IngressRules  []SecurityGroupRule
		EgressRules   []SecurityGroupRule
	}
	SecurityGroupRule struct {
		Description string
		CidrBlocks  []core.IaCValue
		FromPort    int
		Protocol    string
		ToPort      int
		Self        bool
	}
)

const SG_TYPE = "security_group"

// GetSecurityGroup returns the security group if one exists, otherwise creates one, then returns it
func GetSecurityGroup(cfg *config.Application, dag *core.ResourceGraph) *SecurityGroup {
	for _, r := range dag.ListResources() {
		if sg, ok := r.(*SecurityGroup); ok {
			return sg
		}
	}

	vpc := GetVpc(cfg, dag)

	sg := &SecurityGroup{
		Name: cfg.AppName,
		Vpc:  vpc,
	}
	if _, ok := cfg.Import[sg.Id()]; !ok {
		vpcIngressRule := SecurityGroupRule{
			Description: "Allow ingress traffic from ip addresses within the vpc",
			CidrBlocks: []core.IaCValue{
				{Resource: vpc, Property: CIDR_BLOCK_IAC_VALUE},
			},
			FromPort: 0,
			Protocol: "-1",
			ToPort:   0,
		}
		selfIngressRule := SecurityGroupRule{
			Description: "Allow ingress traffic from within the same security group",
			FromPort:    0,
			Protocol:    "-1",
			ToPort:      0,
			Self:        true,
		}
		sg.IngressRules = append(sg.IngressRules, vpcIngressRule, selfIngressRule)

		allOutboundRule := SecurityGroupRule{
			Description: "Allows all outbound IPv4 traffic.",
			FromPort:    0,
			Protocol:    "-1",
			ToPort:      0,
			CidrBlocks: []core.IaCValue{
				{Property: "0.0.0.0/0"},
			},
		}
		sg.EgressRules = append(sg.EgressRules, allOutboundRule)
	}

	dag.AddDependenciesReflect(sg)
	return sg
}

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (sg *SecurityGroup) KlothoConstructRef() []core.AnnotationKey {
	return sg.ConstructsRef
}

// Id returns the id of the cloud resource
func (sg *SecurityGroup) Id() core.ResourceId {
	id := core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     SG_TYPE,
		Name:     sg.Name,
	}
	if sg.Vpc != nil {
		// Realistically, this should only be the case for tests
		id.Namespace = sg.Vpc.Name
	}
	return id
}
