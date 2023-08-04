package knowledgebase

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

var AppRunnerKB = knowledgebase.Build(
	knowledgebase.EdgeBuilder[*resources.AppRunnerService, *resources.EcrImage]{},
	knowledgebase.EdgeBuilder[*resources.AppRunnerService, *resources.DynamodbTable]{
		Configure: func(service *resources.AppRunnerService, table *resources.DynamodbTable, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if service.InstanceRole == nil {
				return fmt.Errorf("cannot configure app runner service %s -> dynamo table %s, missing role", service.Id(), table.Id())
			}
			dag.AddDependency(service.InstanceRole, table)
			// TODO: remove
			if service.EnvironmentVariables == nil {
				service.EnvironmentVariables = map[string]core.IaCValue{}
			}
			service.EnvironmentVariables["TableName"] = core.IaCValue{ResourceId: table.Id(), Property: string(core.KV_DYNAMODB_TABLE_NAME)}
			for _, env := range data.EnvironmentVariables {
				service.EnvironmentVariables[env.GetName()] = core.IaCValue{ResourceId: table.Id(), Property: env.GetValue()}
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.AppRunnerService, *resources.S3Bucket]{
		Configure: func(service *resources.AppRunnerService, bucket *resources.S3Bucket, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if service.InstanceRole == nil {
				return fmt.Errorf("cannot configure app runner service %s -> s3 bucket %s, missing role", service.Id(), bucket.Id())
			}
			dag.AddDependency(service.InstanceRole, bucket)
			for _, env := range data.EnvironmentVariables {
				service.EnvironmentVariables[env.GetName()] = core.IaCValue{ResourceId: bucket.Id(), Property: env.GetValue()}
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.AppRunnerService, *resources.Secret]{
		Configure: func(service *resources.AppRunnerService, secret *resources.Secret, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if service.InstanceRole == nil {
				return fmt.Errorf("cannot configure app runner service %s -> secret %s, missing role", service.Id(), secret.Id())
			}
			dag.AddDependency(service.InstanceRole, secret)
			for _, env := range data.EnvironmentVariables {
				service.EnvironmentVariables[env.GetName()] = core.IaCValue{ResourceId: secret.Id(), Property: env.GetValue()}
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.AppRunnerService, *resources.SesEmailIdentity]{
		Configure: func(service *resources.AppRunnerService, emailIdentity *resources.SesEmailIdentity, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if service.InstanceRole == nil {
				return fmt.Errorf("cannot configure app runner service %s -> s3 bucket %s, missing role", service.Id(), emailIdentity.Id())
			}
			dag.AddDependency(service.InstanceRole, emailIdentity)
			for _, env := range data.EnvironmentVariables {
				service.EnvironmentVariables[env.GetName()] = core.IaCValue{ResourceId: emailIdentity.Id(), Property: env.GetValue()}
			}
			return nil
		},
	},
)
