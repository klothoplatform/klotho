package knowledgebase

import (
	"github.com/klothoplatform/klotho/pkg/core"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

var IamKB = knowledgebase.Build(
	knowledgebase.EdgeBuilder[*resources.RdsProxy, *resources.IamRole]{
		Configure: func(proxy *resources.RdsProxy, role *resources.IamRole, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			role.AssumeRolePolicyDoc = resources.RDS_ASSUME_ROLE_POLICY
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.IamPolicy, *resources.SecretVersion]{
		Configure: func(policy *resources.IamPolicy, secretVersion *resources.SecretVersion, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			secretPolicyDoc := resources.CreateAllowPolicyDocument([]string{"secretsmanager:GetSecretValue"}, []core.IaCValue{{Resource: secretVersion.Secret, Property: resources.ARN_IAC_VALUE}})
			if policy.Policy == nil {
				policy.Policy = secretPolicyDoc
			} else {
				policy.Policy.Statement = append(policy.Policy.Statement, secretPolicyDoc.Statement...)
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.IamRole, *resources.IamPolicy]{
		Configure: func(role *resources.IamRole, policy *resources.IamPolicy, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			role.AddManagedPolicy(core.IaCValue{Resource: policy, Property: resources.ARN_IAC_VALUE})
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.LambdaFunction, *resources.IamRole]{
		Configure: func(lambda *resources.LambdaFunction, role *resources.IamRole, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			role.AssumeRolePolicyDoc = resources.LAMBDA_ASSUMER_ROLE_POLICY
			role.AddAwsManagedPolicies([]string{"arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"})
			return nil
		},
	},
)
