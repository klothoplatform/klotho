package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/engine/classification"
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
	dag.AddResource(instance)
	return nil
}

func (instance *Ec2Instance) MakeOperational(dag *core.ResourceGraph, appName string, classifier classification.Classifier) error {
	if instance.InstanceProfile == nil {
		profiles := core.GetAllDownstreamResourcesOfType[*InstanceProfile](dag, instance)
		if len(profiles) == 0 {
			err := dag.CreateDependencies(instance, map[string]any{
				"InstanceProfile": InstanceProfileCreateParams{
					AppName: appName,
					Refs:    core.BaseConstructSetOf(instance),
					Name:    instance.Name,
				},
			})
			if err != nil {
				return err
			}
		} else if len(profiles) == 1 {
			instance.InstanceProfile = profiles[0]
			dag.AddDependency(instance, instance.InstanceProfile)
		} else {
			return fmt.Errorf("instance %s has more than one instance profile downstream", instance.Id())
		}
	}

	if instance.AMI == nil {
		amis := core.GetAllDownstreamResourcesOfType[*AMI](dag, instance)
		if len(amis) == 0 {
			err := dag.CreateDependencies(instance, map[string]any{
				"AMI": AMICreateParams{
					AppName: appName,
					Refs:    core.BaseConstructSetOf(instance),
					Name:    instance.Name,
				},
			})
			if err != nil {
				return err
			}
		} else if len(amis) == 1 {
			instance.AMI = amis[0]
			dag.AddDependency(instance, instance.AMI)
		} else {
			return fmt.Errorf("instance %s has more than one ami downstream", instance.Id())
		}
	}

	if instance.Subnet == nil {
		vpc, err := getSingleUpstreamVpc(dag, instance)
		if err != nil {
			return err
		}
		subnets := core.GetAllDownstreamResourcesOfType[*Subnet](dag, instance)
		if len(subnets) == 0 {
			subnet, err := core.CreateResource[*Subnet](dag, SubnetCreateParams{
				AppName: appName,
				Refs:    core.BaseConstructSetOf(instance),
				AZ:      "0",
				Type:    PrivateSubnet,
			})
			if err != nil {
				return err
			}
			if vpc != nil {
				dag.AddDependency(subnet, vpc)
			}
			err = subnet.MakeOperational(dag, appName, classifier)
			if err != nil {
				return err
			}
			instance.Subnet = subnet
			dag.AddDependenciesReflect(instance)
		} else if len(subnets) == 1 {
			if subnets[0].Vpc != vpc {
				return fmt.Errorf("instance %s has subnet from vpc which does not correlate to it's vpc downstream", instance.Id())
			}
			instance.Subnet = subnets[0]
			dag.AddDependency(instance, instance.Subnet)
		} else {
			return fmt.Errorf("instance %s has more than one subnet downstream", instance.Id())
		}
	}

	if instance.SecurityGroups == nil {
		sgs, err := getSecurityGroupsOperational(dag, instance, appName)
		if err != nil {
			return err
		}
		instance.SecurityGroups = sgs
		dag.AddDependenciesReflect(instance)
	}
	return nil
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
