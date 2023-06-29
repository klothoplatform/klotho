package knowledgebase

import (
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/infra/kubernetes"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

var EksKB = knowledgebase.Build(
	knowledgebase.EdgeBuilder[*resources.OpenIdConnectProvider, *resources.EksCluster]{
		Configure: func(oidc *resources.OpenIdConnectProvider, cluster *resources.EksCluster, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			oidc.ClientIdLists = []string{"sts.amazonaws.com"}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.EksCluster, *resources.Vpc]{},
	knowledgebase.EdgeBuilder[*resources.EksCluster, *resources.Subnet]{},
	knowledgebase.EdgeBuilder[*resources.EksCluster, *resources.SecurityGroup]{
		Configure: func(cluster *resources.EksCluster, sg *resources.SecurityGroup, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			sg.IngressRules = append(sg.IngressRules, resources.SecurityGroupRule{
				Description: "Allows ingress traffic from the EKS control plane",
				FromPort:    9443,
				Protocol:    "TCP",
				ToPort:      9443,
				CidrBlocks: []*resources.AwsResourceValue{
					{PropertyVal: "0.0.0.0/0"},
				},
			})
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.EksFargateProfile, *resources.Subnet]{},
	knowledgebase.EdgeBuilder[*resources.EksFargateProfile, *resources.EksCluster]{
		Configure: func(profile *resources.EksFargateProfile, cluster *resources.EksCluster, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if len(cluster.GetClustersNodeGroups(dag)) == 0 {
				err := cluster.SetUpDefaultNodeGroup(dag, data.AppName)
				if err != nil {
					return err
				}
			}
			err := cluster.CreateFargateLogging(profile.ConstructsRef, dag)
			return err
		},
	},
	knowledgebase.EdgeBuilder[*resources.EksNodeGroup, *resources.EksCluster]{
		Configure: func(nodeGroup *resources.EksNodeGroup, cluster *resources.EksCluster, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			cluster.CreatePrerequisiteCharts(dag)
			err := cluster.InstallFluentBit(nodeGroup.ConstructsRef, dag)
			if err != nil {
				return err
			}
			if strings.HasSuffix(strings.ToLower(nodeGroup.AmiType), "_gpu") {
				nodeGroup.Cluster.InstallNvidiaDevicePlugin(dag)
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.EksNodeGroup, *resources.Subnet]{},
	knowledgebase.EdgeBuilder[*resources.EksAddon, *resources.EksCluster]{},
	knowledgebase.EdgeBuilder[*kubernetes.HelmChart, *resources.EksCluster]{},
	knowledgebase.EdgeBuilder[*kubernetes.HelmChart, *resources.EksFargateProfile]{},
	knowledgebase.EdgeBuilder[*kubernetes.HelmChart, *resources.EksNodeGroup]{},
	knowledgebase.EdgeBuilder[*kubernetes.HelmChart, *resources.EcrImage]{},
	knowledgebase.EdgeBuilder[*kubernetes.HelmChart, *kubernetes.HelmChart]{},
	knowledgebase.EdgeBuilder[*kubernetes.HelmChart, *kubernetes.KustomizeDirectory]{},
	knowledgebase.EdgeBuilder[*kubernetes.HelmChart, *resources.PrivateDnsNamespace]{},
	knowledgebase.EdgeBuilder[*kubernetes.HelmChart, *resources.TargetGroup]{},
	knowledgebase.EdgeBuilder[*kubernetes.HelmChart, *resources.Region]{},
	knowledgebase.EdgeBuilder[*kubernetes.HelmChart, *resources.Vpc]{},
	knowledgebase.EdgeBuilder[*kubernetes.KustomizeDirectory, *resources.EksCluster]{},
	knowledgebase.EdgeBuilder[*kubernetes.KustomizeDirectory, *resources.EksNodeGroup]{},
	knowledgebase.EdgeBuilder[*kubernetes.Manifest, *kubernetes.KustomizeDirectory]{},
	knowledgebase.EdgeBuilder[*kubernetes.Manifest, *resources.EksCluster]{},
	knowledgebase.EdgeBuilder[*kubernetes.Manifest, *resources.EksFargateProfile]{},
	knowledgebase.EdgeBuilder[*kubernetes.Manifest, *resources.EksNodeGroup]{},
	knowledgebase.EdgeBuilder[*kubernetes.Manifest, *kubernetes.Manifest]{},
	knowledgebase.EdgeBuilder[*kubernetes.Manifest, *resources.Region]{},
	knowledgebase.EdgeBuilder[*kubernetes.HelmChart, *resources.DynamodbTable]{
		Configure: func(chart *kubernetes.HelmChart, table *resources.DynamodbTable, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {

			for _, env := range data.EnvironmentVariables {
				addEnvVarToChart(chart, table, env)
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*kubernetes.HelmChart, *resources.ElasticacheCluster]{
		Configure: func(chart *kubernetes.HelmChart, cluster *resources.ElasticacheCluster, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			for _, env := range data.EnvironmentVariables {
				addEnvVarToChart(chart, cluster, env)
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*kubernetes.HelmChart, *resources.S3Bucket]{
		Configure: func(chart *kubernetes.HelmChart, bucket *resources.S3Bucket, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			for _, env := range data.EnvironmentVariables {
				addEnvVarToChart(chart, bucket, env)
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*kubernetes.HelmChart, *resources.RdsInstance]{
		Configure: func(chart *kubernetes.HelmChart, instance *resources.RdsInstance, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {

			role := GetIamRoleForUnit(chart, data.SourceRef)
			if role == nil {
				return fmt.Errorf("no role found for chart %s and source reference %s in HelmChart to ddb RdsInstance expansion", chart.Id(), data.SourceRef.Id())
			}
			refs := role.ConstructsRef.CloneWith(instance.ConstructsRef)
			inlinePol := resources.NewIamInlinePolicy(fmt.Sprintf("%s-connectionpolicy", instance.Name), refs, instance.GetConnectionPolicyDocument())
			role.InlinePolicies = append(role.InlinePolicies, inlinePol)

			for _, env := range data.EnvironmentVariables {
				addEnvVarToChart(chart, instance, env)
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*kubernetes.HelmChart, *resources.RdsProxy]{
		Configure: func(chart *kubernetes.HelmChart, proxy *resources.RdsProxy, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			role := GetIamRoleForUnit(chart, data.SourceRef)
			if role == nil {
				return fmt.Errorf("no role found for chart %s and source reference %s in HelmChart to ddb RdsProxy expansion", chart.Id(), data.SourceRef.Id())
			}
			upstreamResources := dag.GetUpstreamResources(proxy)
			for _, res := range upstreamResources {
				tg, ok := res.(*resources.RdsProxyTargetGroup)
				if !ok {
					continue
				}
				for _, tgUpstream := range dag.GetDownstreamResources(tg) {
					instance, ok := tgUpstream.(*resources.RdsInstance)
					if !ok {
						continue
					}
					refs := role.ConstructsRef.Clone()
					refs.AddAll(instance.ConstructsRef)
					inlinePol := resources.NewIamInlinePolicy(fmt.Sprintf("%s-connectionpolicy", instance.Name), refs, instance.GetConnectionPolicyDocument())
					role.InlinePolicies = append(role.InlinePolicies, inlinePol)
					dag.AddDependency(role, instance)
				}
			}
			for _, env := range data.EnvironmentVariables {
				addEnvVarToChart(chart, proxy, env)
			}
			return nil
		},
	},
)

func GetIamRoleForUnit(chart *kubernetes.HelmChart, ref core.BaseConstruct) *resources.IamRole {
	rolePlaceholder := kubernetes.GenerateRoleArnPlaceholder(ref.Id().Name)
	for key, val := range chart.Values {
		if rolePlaceholder == key {
			if iacVal, ok := val.(core.IaCValue); ok {
				if role, ok := iacVal.Resource().(*resources.IamRole); ok {
					return role
				}
			}
		}
	}
	return nil
}

func addEnvVarToChart(chart *kubernetes.HelmChart, resource core.Resource, env core.EnvironmentVariable) {
	for _, val := range chart.ProviderValues {
		if val.EnvironmentVariable != nil && env.GetName() == val.EnvironmentVariable.GetName() {
			chart.Values[val.Key] = resources.AwsResourceValue{
				ResourceVal: resource,
				PropertyVal: env.GetValue(),
			}
		}
	}
}
