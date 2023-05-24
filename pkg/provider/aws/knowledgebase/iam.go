package knowledgebase

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
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
			if len(lambda.Subnets) == 0 {
				role.AddAwsManagedPolicies([]string{"arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"})
			} else {
				role.AddAwsManagedPolicies([]string{"arn:aws:iam::aws:policy/service-role/AWSLambdaVPCAccessExecutionRole"})
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.IamRole, *resources.DynamodbTable]{
		Configure: func(role *resources.IamRole, table *resources.DynamodbTable, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			actions := []string{"dynamodb:*"}
			policyResources := []core.IaCValue{
				{Resource: table, Property: resources.ARN_IAC_VALUE},
				{Resource: table, Property: resources.DYNAMODB_TABLE_BACKUP_IAC_VALUE},
				{Resource: table, Property: resources.DYNAMODB_TABLE_INDEX_IAC_VALUE},
				{Resource: table, Property: resources.DYNAMODB_TABLE_EXPORT_IAC_VALUE},
				{Resource: table, Property: resources.DYNAMODB_TABLE_STREAM_IAC_VALUE},
			}
			doc := resources.CreateAllowPolicyDocument(actions, policyResources)
			inlinePol := resources.NewIamInlinePolicy(fmt.Sprintf("%s-dynamodb-policy", table.Name), core.DedupeAnnotationKeys(append(role.ConstructsRef, table.ConstructsRef...)), doc)
			role.InlinePolicies = append(role.InlinePolicies, inlinePol)
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.IamRole, *resources.S3Bucket]{
		Configure: func(role *resources.IamRole, bucket *resources.S3Bucket, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			role.InlinePolicies = append(role.InlinePolicies, resources.NewIamInlinePolicy(
				fmt.Sprintf(`%s-access`, bucket.Name),
				collectionutil.FlattenUnique(role.ConstructsRef, bucket.ConstructsRef),
				resources.CreateAllowPolicyDocument(
					[]string{"s3:*"},
					[]core.IaCValue{
						{Resource: bucket, Property: resources.ARN_IAC_VALUE},
						{Resource: bucket, Property: resources.ALL_BUCKET_DIRECTORY_IAC_VALUE},
					})))
			return nil
		},
	},
)
