package knowledgebase

import (
	"fmt"
	"github.com/klothoplatform/klotho/pkg/core"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

var Ec2KB = knowledgebase.Build(
	knowledgebase.EdgeBuilder[*resources.Ec2Instance, *resources.InstanceProfile]{},
	knowledgebase.EdgeBuilder[*resources.Ec2Instance, *resources.SecurityGroup]{},
	knowledgebase.EdgeBuilder[*resources.Ec2Instance, *resources.Subnet]{},
	knowledgebase.EdgeBuilder[*resources.Ec2Instance, *resources.AMI]{},
	knowledgebase.EdgeBuilder[*resources.Ec2Instance, *resources.DynamodbTable]{
		Configure: func(instance *resources.Ec2Instance, table *resources.DynamodbTable, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			err := checkInstanceForRole(instance, table)
			if err != nil {
				return err
			}
			dag.AddDependency(instance.InstanceProfile.Role, table)
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.Ec2Instance, *resources.ElasticacheCluster]{},
	knowledgebase.EdgeBuilder[*resources.Ec2Instance, *resources.S3Bucket]{
		Configure: func(instance *resources.Ec2Instance, bucket *resources.S3Bucket, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			err := checkInstanceForRole(instance, bucket)
			if err != nil {
				return err
			}
			dag.AddDependency(instance.InstanceProfile.Role, bucket)
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.Ec2Instance, *resources.Secret]{
		Configure: func(instance *resources.Ec2Instance, secret *resources.Secret, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			err := checkInstanceForRole(instance, secret)
			if err != nil {
				return err
			}
			dag.AddDependency(instance.InstanceProfile.Role, secret)
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.Ec2Instance, *resources.RdsInstance]{},
	knowledgebase.EdgeBuilder[*resources.Ec2Instance, *resources.RdsProxy]{},
	knowledgebase.EdgeBuilder[*resources.Ec2Instance, *resources.EfsMountTarget]{
		// Even with this edge configured, a user still needs to mount the EFS volume manually. See: https://docs.aws.amazon.com/efs/latest/ug/wt1-test.html, https://docs.aws.amazon.com/efs/latest/ug/mounting-fs-mount-helper-ec2-linux.html
		Configure: func(instance *resources.Ec2Instance, mountTarget *resources.EfsMountTarget, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if instance.InstanceProfile == nil {
				return fmt.Errorf("cannot configure instance %s -> efs access point %s, missing instance profile", instance.Id(), mountTarget.Id())
			}
			instanceProfile := instance.InstanceProfile
			if instanceProfile.Role == nil {
				return fmt.Errorf("cannot configure instance %s -> efs access point %s, missing instance profile role", instance.Id(), mountTarget.Id())
			}

			efsVpc, err := core.GetSingleDownstreamResourceOfType[*resources.Vpc](dag, mountTarget)
			if err != nil {
				return err
			}
			serviceVpc, _ := core.GetSingleDownstreamResourceOfType[*resources.Vpc](dag, instance)

			if serviceVpc != nil && efsVpc != nil && serviceVpc != efsVpc {
				return fmt.Errorf("instance %s and efs access point %s must be in the same vpc", instance.Id(), mountTarget.Id())
			}

			dag.AddDependency(instanceProfile.Role, mountTarget)

			if serviceVpc == nil {
				dag.AddDependencyWithData(instance, efsVpc, data)
			}

			return nil
		},
	},
)

func checkInstanceForRole(instance *resources.Ec2Instance, dest core.Resource) error {
	if instance.InstanceProfile == nil {
		return fmt.Errorf("cannot configure edge %s -> %s, missing instance profile", instance.Id(), dest.Id())
	} else if instance.InstanceProfile.Role == nil {
		return fmt.Errorf("cannot configure edge %s -> %s, missing instance profile role", instance.Id(), dest.Id())
	}
	return nil
}
