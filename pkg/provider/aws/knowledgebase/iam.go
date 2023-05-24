package knowledgebase

import (
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/infra/kubernetes"
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
	knowledgebase.EdgeBuilder[*resources.EksCluster, *resources.IamRole]{
		Configure: func(cluster *resources.EksCluster, role *resources.IamRole, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			role.AssumeRolePolicyDoc = resources.EKS_ASSUME_ROLE_POLICY
			role.AddAwsManagedPolicies([]string{"arn:aws:iam::aws:policy/AmazonEKSClusterPolicy"})
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.EksFargateProfile, *resources.IamRole]{
		Configure: func(profile *resources.EksFargateProfile, role *resources.IamRole, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			role.AssumeRolePolicyDoc = resources.EKS_FARGATE_ASSUME_ROLE_POLICY
			role.AddAwsManagedPolicies([]string{
				"arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly",
				"arn:aws:iam::aws:policy/AmazonEKSFargatePodExecutionRolePolicy",
			})
			role.InlinePolicies = append(role.InlinePolicies, resources.NewIamInlinePolicy("fargate-pod-execution-policy", profile.ConstructsRef,
				&resources.PolicyDocument{Version: resources.VERSION, Statement: []resources.StatementEntry{
					{
						Effect: "Allow",
						Action: []string{
							"logs:CreateLogStream",
							"logs:CreateLogGroup",
							"logs:DescribeLogStreams",
							"logs:PutLogEvents",
						},
						Resource: []core.IaCValue{{Property: "*"}},
					},
				},
				}))
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.EksNodeGroup, *resources.IamRole]{
		Configure: func(cluster *resources.EksNodeGroup, role *resources.IamRole, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			role.AssumeRolePolicyDoc = resources.EC2_ASSUMER_ROLE_POLICY
			role.AddAwsManagedPolicies([]string{
				"arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy",
				"arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly",
				"arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy",
				"arn:aws:iam::aws:policy/AWSCloudMapFullAccess",
				"arn:aws:iam::aws:policy/CloudWatchAgentServerPolicy",
				"arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore",
			})
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*kubernetes.HelmChart, *resources.IamRole]{
		Expand: func(chart *kubernetes.HelmChart, role *resources.IamRole, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if len(role.ConstructsRef) > 1 {
				return fmt.Errorf("iam role %s must only have one construct ref, but has %d, %s", role.Name, len(role.ConstructsRef), role.ConstructsRef)
			}
			oidc, err := core.CreateResource[*resources.OpenIdConnectProvider](dag, resources.OidcCreateParams{
				AppName:     data.AppName,
				ClusterName: strings.TrimLeft(chart.ClustersProvider.Resource.Id().Name, fmt.Sprintf("%s-", data.AppName)),
				Refs:        role.ConstructsRef,
			})
			dag.AddDependency(role, oidc)
			return err
		},
	},
	knowledgebase.EdgeBuilder[*resources.IamRole, *resources.OpenIdConnectProvider]{
		Configure: func(role *resources.IamRole, oidc *resources.OpenIdConnectProvider, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if len(role.ConstructsRef) > 1 {
				return fmt.Errorf("iam role %s must only have one construct ref, but has %d, %s", role.Name, len(role.ConstructsRef), role.ConstructsRef)
			}
			role.AssumeRolePolicyDoc = resources.GetServiceAccountAssumeRolePolicy(role.ConstructsRef[0].ID, oidc)
			return nil
		},
	},
)
