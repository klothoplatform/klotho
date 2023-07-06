package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/engine/classification"
)

type (
	SecurityGroup struct {
		Name          string
		Vpc           *Vpc
		ConstructsRef core.BaseConstructSet `yaml:"-"`
		IngressRules  []SecurityGroupRule
		EgressRules   []SecurityGroupRule
	}
	SecurityGroupRule struct {
		Description string
		CidrBlocks  []*AwsResourceValue
		FromPort    int
		Protocol    string
		ToPort      int
		Self        bool
	}
)

const SG_TYPE = "security_group"

type SecurityGroupCreateParams struct {
	AppName string
	Refs    core.BaseConstructSet
}

func (sg *SecurityGroup) Create(dag *core.ResourceGraph, params SecurityGroupCreateParams) error {

	sg.Name = params.AppName
	sg.ConstructsRef = params.Refs.Clone()
	existingSG := dag.GetResource(sg.Id())
	if existingSG != nil {
		graphSG := existingSG.(*SecurityGroup)
		graphSG.ConstructsRef.AddAll(params.Refs)
	} else {
		dag.AddResource(sg)
	}
	return nil
}

func (sg *SecurityGroup) MakeOperational(dag *core.ResourceGraph, appName string, classifier classification.Classifier) error {
	sgCopy := *sg
	if sg.Vpc == nil {
		vpc, err := getSingleUpstreamVpc(dag, sg)
		if err != nil {
			return err
		}
		if vpc == nil {
			err := dag.CreateDependencies(sg, map[string]any{
				"Vpc": VpcCreateParams{
					AppName: appName,
					Refs:    core.BaseConstructSetOf(sg),
				},
			})
			if err != nil {
				return err
			}
		} else {
			sg.Vpc = vpc
		}
		err = dag.ReplaceConstruct(&sgCopy, sg)
		if err != nil {
			return err
		}
		dag.AddDependenciesReflect(sg)
	}
	return nil
}

// BaseConstructsRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (sg *SecurityGroup) BaseConstructsRef() core.BaseConstructSet {
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

func (sg *SecurityGroup) Load(namespace string, dag *core.ConstructGraph) error {
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

func (sg *SecurityGroup) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}
