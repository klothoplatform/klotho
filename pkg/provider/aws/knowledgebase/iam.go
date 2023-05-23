package knowledgebase

import (
	"github.com/klothoplatform/klotho/pkg/core"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

var IamKB = knowledgebase.EdgeKB{
	knowledgebase.NewEdge[*resources.RdsProxy, *resources.IamRole](): {
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			role := dest.(*resources.IamRole)
			role.AssumeRolePolicyDoc = resources.RDS_ASSUME_ROLE_POLICY
			return nil
		},
	},
	knowledgebase.NewEdge[*resources.IamPolicy, *resources.SecretVersion](): {
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			policy := source.(*resources.IamPolicy)
			secretVersion := dest.(*resources.SecretVersion)
			secretPolicyDoc := resources.CreateAllowPolicyDocument([]string{"secretsmanager:GetSecretValue"}, []core.IaCValue{{Resource: secretVersion.Secret, Property: resources.ARN_IAC_VALUE}})
			if policy.Policy == nil {
				policy.Policy = secretPolicyDoc
			} else {
				policy.Policy.Statement = append(policy.Policy.Statement, secretPolicyDoc.Statement...)
			}
			return nil
		},
	},
	knowledgebase.NewEdge[*resources.IamRole, *resources.IamPolicy](): {
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			policy := dest.(*resources.IamPolicy)
			role := source.(*resources.IamRole)
			role.AddManagedPolicy(core.IaCValue{Resource: policy, Property: resources.ARN_IAC_VALUE})
			return nil
		},
	},
	knowledgebase.NewEdge[*resources.LambdaFunction, *resources.IamRole](): {
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			role := dest.(*resources.IamRole)
			role.AssumeRolePolicyDoc = resources.LAMBDA_ASSUMER_ROLE_POLICY
			role.AddAwsManagedPolicies([]string{"arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"})
			return nil
		},
	},
}
