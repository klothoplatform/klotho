package knowledgebase

import (
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/core"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	kubernetes "github.com/klothoplatform/klotho/pkg/provider/kubernetes/resources"
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
				CidrBlocks: []core.IaCValue{
					{Property: "0.0.0.0/0"},
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
			err := cluster.CreateFargateLogging(profile.ConstructRefs, dag)
			return err
		},
	},
	knowledgebase.EdgeBuilder[*resources.EksNodeGroup, *resources.EksCluster]{
		Configure: func(nodeGroup *resources.EksNodeGroup, cluster *resources.EksCluster, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			cluster.CreatePrerequisiteCharts(dag)
			err := cluster.InstallFluentBit(nodeGroup.ConstructRefs, dag)
			if err != nil {
				return err
			}
			if strings.HasSuffix(strings.ToLower(nodeGroup.AmiType), "_gpu") {
				nodeGroup.Cluster.InstallNvidiaDevicePlugin(dag)
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*kubernetes.ServiceAccount, *resources.IamRole]{},
	knowledgebase.EdgeBuilder[*resources.EksNodeGroup, *resources.Subnet]{},
	knowledgebase.EdgeBuilder[*resources.EksAddon, *resources.EksCluster]{},
	knowledgebase.EdgeBuilder[*kubernetes.HelmChart, *resources.EksCluster]{},
	knowledgebase.EdgeBuilder[*kubernetes.Pod, *resources.EksCluster]{},
	knowledgebase.EdgeBuilder[*kubernetes.Deployment, *resources.EksCluster]{},
	knowledgebase.EdgeBuilder[*kubernetes.Service, *resources.EksCluster]{},
	knowledgebase.EdgeBuilder[*kubernetes.ServiceAccount, *resources.EksCluster]{},
	knowledgebase.EdgeBuilder[*kubernetes.TargetGroupBinding, *resources.EksCluster]{},
	knowledgebase.EdgeBuilder[*kubernetes.ServiceExport, *resources.EksCluster]{},
	knowledgebase.EdgeBuilder[*kubernetes.HorizontalPodAutoscaler, *resources.EksCluster]{},
	knowledgebase.EdgeBuilder[*kubernetes.Pod, *resources.EksFargateProfile]{},
	knowledgebase.EdgeBuilder[*kubernetes.Deployment, *resources.EksFargateProfile]{},
	knowledgebase.EdgeBuilder[*kubernetes.Pod, *resources.EksNodeGroup]{},
	knowledgebase.EdgeBuilder[*kubernetes.Deployment, *resources.EksNodeGroup]{},

	knowledgebase.EdgeBuilder[*kubernetes.HelmChart, *resources.EksFargateProfile]{},
	knowledgebase.EdgeBuilder[*kubernetes.HelmChart, *resources.EksNodeGroup]{},
	knowledgebase.EdgeBuilder[*kubernetes.HelmChart, *resources.EcrImage]{},
	knowledgebase.EdgeBuilder[*kubernetes.HelmChart, *kubernetes.HelmChart]{},
	knowledgebase.EdgeBuilder[*kubernetes.HelmChart, *kubernetes.KustomizeDirectory]{},
	knowledgebase.EdgeBuilder[*kubernetes.HelmChart, *resources.PrivateDnsNamespace]{},
	knowledgebase.EdgeBuilder[*resources.TargetGroup, *kubernetes.TargetGroupBinding]{
		DeploymentOrderReversed: true,
	},
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

	knowledgebase.EdgeBuilder[*kubernetes.Pod, *resources.DynamodbTable]{
		Configure: func(pod *kubernetes.Pod, table *resources.DynamodbTable, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			role, err := GetPodServiceAccountRole(pod, dag)
			if err != nil {
				return err
			}
			dag.AddDependency(role, table)
			for _, env := range data.EnvironmentVariables {
				err := pod.AddEnvVar(core.IaCValue{ResourceId: table.Id(), Property: env.GetValue()}, env.GetName())
				if err != nil {
					return err
				}
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*kubernetes.Deployment, *resources.DynamodbTable]{
		Configure: func(deployment *kubernetes.Deployment, table *resources.DynamodbTable, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			role, err := GetDeploymentServiceAccountRole(deployment, dag)
			if err != nil {
				return err
			}
			dag.AddDependency(role, table)
			for _, env := range data.EnvironmentVariables {
				err := deployment.AddEnvVar(core.IaCValue{ResourceId: table.Id(), Property: env.GetValue()}, env.GetName())
				if err != nil {
					return err
				}
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*kubernetes.Pod, *resources.ElasticacheCluster]{
		Configure: func(pod *kubernetes.Pod, cluster *resources.ElasticacheCluster, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			for _, env := range data.EnvironmentVariables {
				err := pod.AddEnvVar(core.IaCValue{ResourceId: cluster.Id(), Property: env.GetValue()}, env.GetName())
				if err != nil {
					return err
				}
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*kubernetes.Deployment, *resources.ElasticacheCluster]{
		Configure: func(deployment *kubernetes.Deployment, cluster *resources.ElasticacheCluster, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			for _, env := range data.EnvironmentVariables {
				err := deployment.AddEnvVar(core.IaCValue{ResourceId: cluster.Id(), Property: env.GetValue()}, env.GetName())
				if err != nil {
					return err
				}
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*kubernetes.Pod, *resources.S3Bucket]{
		Configure: func(pod *kubernetes.Pod, bucket *resources.S3Bucket, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			role, err := GetPodServiceAccountRole(pod, dag)
			if err != nil {
				return err
			}
			dag.AddDependency(role, bucket)
			for _, env := range data.EnvironmentVariables {
				err := pod.AddEnvVar(core.IaCValue{ResourceId: bucket.Id(), Property: env.GetValue()}, env.GetName())
				if err != nil {
					return err
				}
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*kubernetes.Deployment, *resources.S3Bucket]{
		Configure: func(deployment *kubernetes.Deployment, bucket *resources.S3Bucket, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			role, err := GetDeploymentServiceAccountRole(deployment, dag)
			if err != nil {
				return err
			}
			dag.AddDependency(role, bucket)
			for _, env := range data.EnvironmentVariables {
				err := deployment.AddEnvVar(core.IaCValue{ResourceId: bucket.Id(), Property: env.GetValue()}, env.GetName())
				if err != nil {
					return err
				}
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*kubernetes.Pod, *resources.RdsInstance]{
		Configure: func(pod *kubernetes.Pod, instance *resources.RdsInstance, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			role, err := GetPodServiceAccountRole(pod, dag)
			if err != nil {
				return err
			}
			dag.AddDependency(role, instance)
			for _, env := range data.EnvironmentVariables {
				err := pod.AddEnvVar(core.IaCValue{ResourceId: instance.Id(), Property: env.GetValue()}, env.GetName())
				if err != nil {
					return err
				}
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*kubernetes.Deployment, *resources.RdsInstance]{
		Configure: func(deployment *kubernetes.Deployment, instance *resources.RdsInstance, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			role, err := GetDeploymentServiceAccountRole(deployment, dag)
			if err != nil {
				return err
			}
			dag.AddDependency(role, instance)
			for _, env := range data.EnvironmentVariables {
				err := deployment.AddEnvVar(core.IaCValue{ResourceId: instance.Id(), Property: env.GetValue()}, env.GetName())
				if err != nil {
					return err
				}
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*kubernetes.Pod, *resources.RdsProxy]{
		Configure: func(pod *kubernetes.Pod, proxy *resources.RdsProxy, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			role, err := GetPodServiceAccountRole(pod, dag)
			if err != nil {
				return err
			}
			dag.AddDependency(role, proxy)
			for _, env := range data.EnvironmentVariables {
				err := pod.AddEnvVar(core.IaCValue{ResourceId: proxy.Id(), Property: env.GetValue()}, env.GetName())
				if err != nil {
					return err
				}
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*kubernetes.Deployment, *resources.RdsProxy]{
		Configure: func(deployment *kubernetes.Deployment, proxy *resources.RdsProxy, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			role, err := GetDeploymentServiceAccountRole(deployment, dag)
			if err != nil {
				return err
			}
			dag.AddDependency(role, proxy)
			for _, env := range data.EnvironmentVariables {
				err := deployment.AddEnvVar(core.IaCValue{ResourceId: proxy.Id(), Property: env.GetValue()}, env.GetName())
				if err != nil {
					return err
				}
			}
			return nil
		},
	},
)

func GetPodServiceAccountRole(pod *kubernetes.Pod, dag *core.ResourceGraph) (*resources.IamRole, error) {
	sa := pod.GetServiceAccount(dag)
	if sa == nil {
		return nil, fmt.Errorf("no service account found for pod %s in Pod during expansion", pod.Id())
	}
	role, err := resources.GetServiceAccountRole(sa, dag)
	if err != nil {
		return nil, err
	}
	return role, nil
}

func GetDeploymentServiceAccountRole(deployment *kubernetes.Deployment, dag *core.ResourceGraph) (*resources.IamRole, error) {
	sa := deployment.GetServiceAccount(dag)
	if sa == nil {
		return nil, fmt.Errorf("no service account found for pod %s in Pod during expansion", deployment.Id())
	}
	role, err := resources.GetServiceAccountRole(sa, dag)
	if err != nil {
		return nil, err
	}
	return role, nil
}
