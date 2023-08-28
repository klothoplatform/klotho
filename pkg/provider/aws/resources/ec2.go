package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

const (
	EC2_INSTANCE_TYPE = "ec2_instance"
	AMI_TYPE          = "ami"
)

type (
	Ec2Instance struct {
		Name            string
		ConstructRefs   construct.BaseConstructSet `yaml:"-"`
		InstanceProfile *InstanceProfile
		SecurityGroups  []*SecurityGroup
		Subnet          *Subnet
		AMI             *AMI
		InstanceType    string
	}

	AMI struct {
		Name          string
		ConstructRefs construct.BaseConstructSet `yaml:"-"`
	}
)

type Ec2InstanceCreateParams struct {
	Name    string
	AppName string
	Refs    construct.BaseConstructSet
}

func (instance *Ec2Instance) Create(dag *construct.ResourceGraph, params Ec2InstanceCreateParams) error {
	instance.Name = aws.Ec2InstanceSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	instance.ConstructRefs = params.Refs.Clone()

	existingInstance, found := construct.GetResource[*Ec2Instance](dag, instance.Id())
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
	Refs    construct.BaseConstructSet
}

func (ami *AMI) Create(dag *construct.ResourceGraph, params AMICreateParams) error {
	ami.Name = aws.Ec2InstanceSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	ami.ConstructRefs = params.Refs.Clone()

	existingAMI, found := construct.GetResource[*AMI](dag, ami.Id())
	if found {
		existingAMI.ConstructRefs.AddAll(params.Refs)
		return nil
	}
	dag.AddResource(ami)
	return nil
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (instance *Ec2Instance) BaseConstructRefs() construct.BaseConstructSet {
	return instance.ConstructRefs
}

// Id returns the id of the cloud resource
func (instance *Ec2Instance) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     EC2_INSTANCE_TYPE,
		Name:     instance.Name,
	}
}

func (instance *Ec2Instance) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream:     true,
		RequiresNoDownstream:   true,
		RequiresExplicitDelete: true,
	}
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (ami *AMI) BaseConstructRefs() construct.BaseConstructSet {
	return ami.ConstructRefs
}

// Id returns the id of the cloud resource
func (ami *AMI) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     AMI_TYPE,
		Name:     ami.Name,
	}
}

func (ami *AMI) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream: true,
	}
}
