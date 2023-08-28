package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/construct"
)

type (
	SecurityGroup struct {
		Name          string
		Vpc           *Vpc
		ConstructRefs construct.BaseConstructSet `yaml:"-"`
		IngressRules  []SecurityGroupRule
		EgressRules   []SecurityGroupRule
	}
	SecurityGroupRule struct {
		Description string
		CidrBlocks  []construct.IaCValue
		FromPort    int
		Protocol    string
		ToPort      int
		Self        bool
	}
)

const SG_TYPE = "security_group"

type SecurityGroupCreateParams struct {
	AppName string
	Refs    construct.BaseConstructSet
}

func (sg *SecurityGroup) Create(dag *construct.ResourceGraph, params SecurityGroupCreateParams) error {

	sg.Name = params.AppName
	sg.ConstructRefs = params.Refs.Clone()
	existingSG := dag.GetResource(sg.Id())
	if existingSG != nil {
		graphSG := existingSG.(*SecurityGroup)
		graphSG.ConstructRefs.AddAll(params.Refs)
	} else {
		dag.AddResource(sg)
	}
	return nil
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (sg *SecurityGroup) BaseConstructRefs() construct.BaseConstructSet {
	return sg.ConstructRefs
}

// Id returns the id of the cloud resource
func (sg *SecurityGroup) Id() construct.ResourceId {
	id := construct.ResourceId{
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

func (sg *SecurityGroup) Load(namespace string, dag *construct.ConstructGraph) error {
	namespacedVpc := &Vpc{Name: namespace}
	vpc := dag.GetConstruct(namespacedVpc.Id())
	if vpc == nil {
		return fmt.Errorf("cannot load subnet with name %s because namespace vpc %s does not exist", sg.Name, namespace)
	}
	if vpc, ok := vpc.(*Vpc); !ok {
		return fmt.Errorf("cannot load subnet with name %s because namespace vpc %s is not a vpc", sg.Name, namespace)
	} else {
		sg.Vpc = vpc
	}
	return nil
}

func (sg *SecurityGroup) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream: true,
	}
}

func (rule *SecurityGroupRule) Equals(other SecurityGroupRule) bool {
	if rule.Description != other.Description {
		return false
	}
	if rule.FromPort != other.FromPort {
		return false
	}
	if rule.Protocol != other.Protocol {
		return false
	}
	if rule.ToPort != other.ToPort {
		return false
	}
	if rule.Self != other.Self {
		return false
	}
	if len(rule.CidrBlocks) != len(other.CidrBlocks) {
		return false
	}
	for i, cidrBlock := range rule.CidrBlocks {
		if cidrBlock != other.CidrBlocks[i] {
			return false
		}
	}
	return true
}
