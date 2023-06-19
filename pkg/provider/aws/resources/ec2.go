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
		ConstructsRef   core.BaseConstructSet `yaml:"-"`
		InstanceProfile *InstanceProfile
		SecurityGroups  []*SecurityGroup
		Subnet          *Subnet
		AMI             *AMI
		InstanceType    string
	}

	AMI struct {
		Name          string
		ConstructsRef core.BaseConstructSet `yaml:"-"`
	}
)

type Ec2InstanceCreateParams struct {
	Name    string
	AppName string
	Refs    core.BaseConstructSet
}

func (instance *Ec2Instance) Create(dag *core.ResourceGraph, params Ec2InstanceCreateParams) error {
	instance.Name = aws.Ec2InstanceSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	instance.ConstructsRef = params.Refs.Clone()

	existingInstance, found := core.GetResource[*Ec2Instance](dag, instance.Id())
	if found {
		existingInstance.ConstructsRef.AddAll(params.Refs)
		return nil
	}
	instance.SecurityGroups = make([]*SecurityGroup, 1)
	return dag.CreateDependencies(instance, map[string]any{
		"InstanceProfile": InstanceProfileCreateParams{
			AppName: params.AppName,
			Name:    params.Name,
			Refs:    params.Refs,
		},
		"SecurityGroups": []*SecurityGroupCreateParams{
			{
				AppName: params.AppName,
				Refs:    params.Refs,
			},
		},
		"Subnet": SubnetCreateParams{
			AppName: params.AppName,
			Refs:    params.Refs,
			AZ:      "0",
			Type:    PrivateSubnet,
		},
		"AMI": params,
	})
}

type Ec2InstanceConfigureParams struct {
	InstanceType string
}

type AMICreateParams struct {
	Name    string
	AppName string
	Refs    core.BaseConstructSet
}

func (ami *AMI) Create(dag *core.ResourceGraph, params AMICreateParams) error {
	ami.Name = aws.Ec2InstanceSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	ami.ConstructsRef = params.Refs.Clone()

	existingAMI, found := core.GetResource[*AMI](dag, ami.Id())
	if found {
		existingAMI.ConstructsRef.AddAll(params.Refs)
		return nil
	}
	dag.AddResource(ami)
	return nil
}

func (instance *Ec2Instance) Configure(params Ec2InstanceConfigureParams) error {
	instance.InstanceType = "t3.medium"
	if params.InstanceType != "" {
		instance.InstanceType = params.InstanceType
	}
	return nil
}

// BaseConstructsRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (instance *Ec2Instance) BaseConstructsRef() core.BaseConstructSet {
	return instance.ConstructsRef
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

// BaseConstructsRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (ami *AMI) BaseConstructsRef() core.BaseConstructSet {
	return ami.ConstructsRef
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
