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
			secretPolicyDoc := resources.CreateAllowPolicyDocument([]string{"secretsmanager:DescribeSecret", "secretsmanager:GetSecretValue"}, []*resources.AwsResourceValue{{ResourceVal: secret, PropertyVal: resources.ARN_IAC_VALUE}})
			inlinePol := resources.NewIamInlinePolicy(fmt.Sprintf("%s-secretpolicy", secret.Name), role.ConstructsRef.CloneWith(secret.ConstructsRef), secretPolicyDoc)
			role.InlinePolicies = append(role.InlinePolicies, inlinePol)
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.IamPolicy, *resources.Secret]{
		Configure: func(policy *resources.IamPolicy, secret *resources.Secret, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			policy.AddPolicyDocument(resources.CreateAllowPolicyDocument([]string{"secretsmanager:DescribeSecret", "secretsmanager:GetSecretValue"}, []*resources.AwsResourceValue{{ResourceVal: secret, PropertyVal: resources.ARN_IAC_VALUE}}))
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.IamRole, *resources.IamPolicy]{
		Configure: func(role *resources.IamRole, policy *resources.IamPolicy, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			role.AddManagedPolicy(&resources.AwsResourceValue{ResourceVal: policy, PropertyVal: resources.ARN_IAC_VALUE})
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
	knowledgebase.EdgeBuilder[*resources.EcsTaskDefinition, *resources.IamRole]{
		Configure: func(taskDef *resources.EcsTaskDefinition, role *resources.IamRole, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			role.AssumeRolePolicyDoc = resources.ECS_ASSUMER_ROLE_POLICY
			role.AddAwsManagedPolicies([]string{"arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"})
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.IamRole, *resources.DynamodbTable]{
		Configure: func(role *resources.IamRole, table *resources.DynamodbTable, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			actions := []string{"dynamodb:*"}
			policyResources := []*resources.AwsResourceValue{
				{ResourceVal: table, PropertyVal: resources.ARN_IAC_VALUE},
				{ResourceVal: table, PropertyVal: resources.DYNAMODB_TABLE_BACKUP_IAC_VALUE},
				{ResourceVal: table, PropertyVal: resources.DYNAMODB_TABLE_INDEX_IAC_VALUE},
				{ResourceVal: table, PropertyVal: resources.DYNAMODB_TABLE_EXPORT_IAC_VALUE},
				{ResourceVal: table, PropertyVal: resources.DYNAMODB_TABLE_STREAM_IAC_VALUE},
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
						Resource: []*resources.AwsResourceValue{{PropertyVal: "*"}},
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
				ClusterName: strings.TrimLeft(chart.ClustersProvider.Resource().Id().Name, fmt.Sprintf("%s-", data.AppName)),
				Refs:        role.ConstructsRef.Clone(),
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
				ClusterName: strings.TrimLeft(manifest.ClustersProvider.Resource().Id().Name, fmt.Sprintf("%s-", data.AppName)),
				Refs:        role.ConstructsRef.Clone(),
			})
			dag.AddDependency(role, oidc)
			return err
		},
	},
	knowledgebase.EdgeBuilder[*resources.IamRole, *resources.OpenIdConnectProvider]{
		Configure: func(role *resources.IamRole, oidc *resources.OpenIdConnectProvider, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if strings.Contains(role.Name, "alb-controller") {
				role.AssumeRolePolicyDoc = resources.GetServiceAccountAssumeRolePolicy("aws-load-balancer-controller", oidc)
				return nil
			}
			if len(role.ConstructsRef) > 1 {
				return fmt.Errorf("iam role %s must only have one construct ref, but has %d, %s", role.Name, len(role.ConstructsRef), role.ConstructsRef)
			}
			var ref core.ResourceId
			for cons := range role.ConstructsRef {
				ref = cons
			}
			role.AssumeRolePolicyDoc = resources.GetServiceAccountAssumeRolePolicy(ref.Name, oidc)
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
					[]*resources.AwsResourceValue{
						{ResourceVal: bucket, PropertyVal: resources.ARN_IAC_VALUE},
						{ResourceVal: bucket, PropertyVal: resources.ALL_BUCKET_DIRECTORY_IAC_VALUE},
					})))
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.IamPolicy, *resources.LambdaFunction]{
		Configure: func(policy *resources.IamPolicy, function *resources.LambdaFunction, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			policy.AddPolicyDocument(resources.CreateAllowPolicyDocument([]string{"lambda:InvokeFunction"}, []*resources.AwsResourceValue{{ResourceVal: function, PropertyVal: resources.ARN_IAC_VALUE}}))
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.IamRole, *resources.RdsInstance]{
		Configure: func(role *resources.IamRole, instance *resources.RdsInstance, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			inlinePol := resources.NewIamInlinePolicy(fmt.Sprintf("%s-connectionpolicy", instance.Name),
				role.ConstructsRef.CloneWith(instance.ConstructsRef), instance.GetConnectionPolicyDocument())
			role.InlinePolicies = append(role.InlinePolicies, inlinePol)
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.IamRole, *resources.RdsProxy]{},
	knowledgebase.EdgeBuilder[*resources.RolePolicyAttachment, *resources.IamRole]{},
	knowledgebase.EdgeBuilder[*resources.RolePolicyAttachment, *resources.IamPolicy]{},
	knowledgebase.EdgeBuilder[*resources.IamPolicy, *resources.PrivateDnsNamespace]{
		Configure: func(policy *resources.IamPolicy, namespace *resources.PrivateDnsNamespace, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			policy.AddPolicyDocument(resources.CreateAllowPolicyDocument([]string{"servicediscovery:DiscoverInstances"}, []*resources.AwsResourceValue{{PropertyVal: core.ALL_RESOURCES_IAC_VALUE}}))
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.InstanceProfile, *resources.IamRole]{
		Configure: func(source *resources.InstanceProfile, destination *resources.IamRole, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			inlinePolicy := resources.NewIamInlinePolicy(fmt.Sprintf("%s-instanceProfilePolicy", source.Name), source.ConstructsRef.CloneWith(destination.ConstructsRef),
				&resources.PolicyDocument{
					Version: resources.VERSION,
					Statement: resources.CreateAllowPolicyDocument([]string{
						"iam:ListInstanceProfiles",
						"ec2:Describe*",
						"ec2:Search*",
						"ec2:Get*",
					}, []*resources.AwsResourceValue{{PropertyVal: "*"}}).Statement,
				},
			)
			inlinePolicy.Policy.Statement = append(inlinePolicy.Policy.Statement, resources.StatementEntry{
				Effect:   "Allow",
				Action:   []string{"iam:PassRole"},
				Resource: []*resources.AwsResourceValue{{PropertyVal: "*"}},
				Condition: &resources.Condition{
					StringEquals: map[*resources.AwsResourceValue]string{
						{PropertyVal: "iam:PassedToService"}: "ec2.amazonaws.com",
					},
				},
			})
			destination.InlinePolicies = append(destination.InlinePolicies, inlinePolicy)
			return nil
		},
	},
)
