package knowledgebase

import (
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
		Expand: func(instance *resources.Ec2Instance, table *resources.DynamodbTable, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			dag.AddDependency(instance.InstanceProfile.Role, table)
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.Ec2Instance, *resources.ElasticacheCluster]{},
	knowledgebase.EdgeBuilder[*resources.Ec2Instance, *resources.S3Bucket]{
		Expand: func(instance *resources.Ec2Instance, bucket *resources.S3Bucket, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			dag.AddDependency(instance.InstanceProfile.Role, bucket)
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.Ec2Instance, *resources.Secret]{
		Expand: func(instance *resources.Ec2Instance, secret *resources.Secret, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			dag.AddDependency(instance.InstanceProfile.Role, secret)
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.Ec2Instance, *resources.RdsInstance]{},
	knowledgebase.EdgeBuilder[*resources.Ec2Instance, *resources.RdsProxy]{
		ValidDestinations: []core.Resource{&resources.RdsInstance{}},
	},
)
