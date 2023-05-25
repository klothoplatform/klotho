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
	knowledgebase.EdgeBuilder[*resources.IamRole, *resources.Secret]{
		Configure: func(role *resources.IamRole, secret *resources.Secret, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			secretPolicyDoc := resources.CreateAllowPolicyDocument([]string{"secretsmanager:DescribeSecret", "secretsmanager:GetSecretValue"}, []core.IaCValue{{Resource: secret, Property: resources.ARN_IAC_VALUE}})
			inlinePol := resources.NewIamInlinePolicy(fmt.Sprintf("%s-secretpolicy", secret.Name), role.ConstructsRef.CloneWith(secret.ConstructsRef), secretPolicyDoc)
			role.InlinePolicies = append(role.InlinePolicies, inlinePol)
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.IamPolicy, *resources.Secret]{
		Configure: func(policy *resources.IamPolicy, secret *resources.Secret, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			secretPolicyDoc := resources.CreateAllowPolicyDocument([]string{"secretsmanager:DescribeSecret", "secretsmanager:GetSecretValue"}, []core.IaCValue{{Resource: secret, Property: resources.ARN_IAC_VALUE}})
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
			inlinePol := resources.NewIamInlinePolicy(fmt.Sprintf("%s-dynamodb-policy", table.Name), role.ConstructsRef.CloneWith(table.ConstructsRef), doc)
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
	knowledgebase.EdgeBuilder[*kubernetes.Manifest, *resources.IamRole]{
		Expand: func(manifest *kubernetes.Manifest, role *resources.IamRole, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			// For certain scenarios (like the alb controller) where we arent creating a service account for a unit derived in klotho, we have no understanding of what that service account is.
			// Once we make specific kubernetes objects resources we could have that understanding
			if role.AssumeRolePolicyDoc != nil {
				return nil
			}
			if len(role.ConstructsRef) > 1 {
				return fmt.Errorf("iam role %s must only have one construct ref, but has %d, %s", role.Name, len(role.ConstructsRef), role.ConstructsRef)
			}
			oidc, err := core.CreateResource[*resources.OpenIdConnectProvider](dag, resources.OidcCreateParams{
				AppName:     data.AppName,
				ClusterName: strings.TrimLeft(manifest.ClustersProvider.Resource.Id().Name, fmt.Sprintf("%s-", data.AppName)),
				Refs:        role.ConstructsRef,
			})
			dag.AddDependency(role, oidc)
			return err
		},
	},
	knowledgebase.EdgeBuilder[*resources.IamRole, *resources.OpenIdConnectProvider]{
		Configure: func(role *resources.IamRole, oidc *resources.OpenIdConnectProvider, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			ref, oneRef := role.ConstructsRef.GetSingle()
			if !oneRef {
				return fmt.Errorf("iam role %s must only have one construct ref, but has %d, %s", role.Name, len(role.ConstructsRef), role.ConstructsRef)
			}
			role.AssumeRolePolicyDoc = resources.GetServiceAccountAssumeRolePolicy(ref.ID, oidc)
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.IamRole, *resources.S3Bucket]{
		Configure: func(role *resources.IamRole, bucket *resources.S3Bucket, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			role.InlinePolicies = append(role.InlinePolicies, resources.NewIamInlinePolicy(
				fmt.Sprintf(`%s-access`, bucket.Name),
				role.ConstructsRef.CloneWith(bucket.ConstructsRef),
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
