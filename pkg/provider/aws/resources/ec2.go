package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

const (
	EC2_INSTANCE_TYPE = "ec2_instance"
	AMI_TYPE          = "ami"
)

type (
	Ec2Instance struct {
		Name            string
		ConstructRefs   core.BaseConstructSet `yaml:"-"`
		InstanceProfile *InstanceProfile
		SecurityGroups  []*SecurityGroup
		Subnet          *Subnet
		AMI             *AMI
		InstanceType    string
	}

	AMI struct {
		Name          string
		ConstructRefs core.BaseConstructSet `yaml:"-"`
	}
)

type Ec2InstanceCreateParams struct {
	Name    string
	AppName string
	Refs    core.BaseConstructSet
}

func (instance *Ec2Instance) Create(dag *core.ResourceGraph, params Ec2InstanceCreateParams) error {
	instance.Name = aws.Ec2InstanceSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	instance.ConstructRefs = params.Refs.Clone()

	existingInstance, found := core.GetResource[*Ec2Instance](dag, instance.Id())
	if found {
		existingInstance.ConstructRefs.AddAll(params.Refs)
		return nil
	}
	dag.AddResource(instance)
	return nil
}

type AMICreateParams struct {
	Name    string
	AppName string
	Refs    core.BaseConstructSet
}

func (ami *AMI) Create(dag *core.ResourceGraph, params AMICreateParams) error {
	ami.Name = aws.Ec2InstanceSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	ami.ConstructRefs = params.Refs.Clone()

	existingAMI, found := core.GetResource[*AMI](dag, ami.Id())
	if found {
		existingAMI.ConstructRefs.AddAll(params.Refs)
		return nil
	}
	dag.AddResource(ami)
	return nil
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (instance *Ec2Instance) BaseConstructRefs() core.BaseConstructSet {
	return instance.ConstructRefs
}

// Id returns the id of the cloud resource
func (instance *Ec2Instance) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     EC2_INSTANCE_TYPE,
		Name:     instance.Name,
	}
}

func (instance *Ec2Instance) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream:     true,
		RequiresNoDownstream:   true,
		RequiresExplicitDelete: true,
	}
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (ami *AMI) BaseConstructRefs() core.BaseConstructSet {
	return ami.ConstructRefs
}

// Id returns the id of the cloud resource
func (ami *AMI) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     AMI_TYPE,
		Name:     ami.Name,
	}
}

func (ami *AMI) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}
