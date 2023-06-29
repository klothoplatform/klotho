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
)

func checkInstanceForRole(instance *resources.Ec2Instance, dest core.Resource) error {
	if instance.InstanceProfile == nil {
		return fmt.Errorf("cannot configure edge %s -> %s, missing instance profile", instance.Id(), dest.Id())
	} else if instance.InstanceProfile.Role == nil {
		return fmt.Errorf("cannot configure edge %s -> %s, missing instance profile role", instance.Id(), dest.Id())
	}
	return nil
}
