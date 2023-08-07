package knowledgebase

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/aws-load-balancer-controller/apis/elbv2/v1beta1"
	"strings"

	"github.com/klothoplatform/klotho/pkg/core"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	kubernetes "github.com/klothoplatform/klotho/pkg/provider/kubernetes/resources"
)

var EksKB = knowledgebase.Build(
	knowledgebase.EdgeBuilder[*resources.OpenIdConnectProvider, *resources.EksCluster]{
		Configure: func(oidc *resources.OpenIdConnectProvider, cluster *resources.EksCluster, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			oidc.Cluster = cluster
			oidc.ClientIdLists = []string{"sts.amazonaws.com"}

			if oidc.Region == nil {
				oidc.Region = resources.NewRegion()
			}
			dag.AddDependenciesReflect(oidc)
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.EksCluster, *resources.Vpc]{},
	knowledgebase.EdgeBuilder[*resources.EksCluster, *resources.Subnet]{},
	knowledgebase.EdgeBuilder[*kubernetes.Kubeconfig, *resources.EksCluster]{},
	knowledgebase.EdgeBuilder[*resources.EksCluster, *kubernetes.Kubeconfig]{
		DeploymentOrderReversed: true,
	},
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
	knowledgebase.EdgeBuilder[*kubernetes.ServiceAccount, *resources.IamRole]{
		// Links the service account to the IAM role using IRSA: https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html
		Configure: func(sa *kubernetes.ServiceAccount, role *resources.IamRole, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if sa.Object == nil {
				return fmt.Errorf("service account %s has no object", sa.Name)
			}

			value := resources.GenerateRoleArnPlaceholder(role.Name)
			roleArnPlaceholder := fmt.Sprintf("{{ .Values.%s }}", value)

			if sa.Object.Annotations == nil {
				sa.Object.Annotations = make(map[string]string)
			}
			sa.Object.Annotations["eks.amazonaws.com/role-arn"] = roleArnPlaceholder

			if sa.Transformations == nil {
				sa.Transformations = make(map[string]core.IaCValue)
			}
			sa.Transformations[value] = core.IaCValue{ResourceId: role.Id(), Property: resources.ID_IAC_VALUE}

			// Sets the role's AssumeRolePolicyDocument to allow the service account to assume the role
			oidc, err := core.CreateResource[*resources.OpenIdConnectProvider](dag, resources.OidcCreateParams{
				AppName:     data.AppName,
				ClusterName: sa.Cluster.Name,
				Refs:        core.BaseConstructSetOf(sa),
			})
			if err != nil {
				return err
			}
			assumeRolePolicy := resources.GetServiceAccountAssumeRolePolicy(sa.Object.Name, oidc)
			role.AssumeRolePolicyDoc = assumeRolePolicy
			dag.AddDependenciesReflect(role)

			return nil
		},
	},
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
	knowledgebase.EdgeBuilder[*kubernetes.Pod, *resources.EcrImage]{
		Configure: func(pod *kubernetes.Pod, image *resources.EcrImage, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if pod.Object == nil {
				return fmt.Errorf("pod %s has no object", pod.Name)
			}
			value := resources.GenerateImagePlaceholder(image.Name)
			imagePlaceholder := fmt.Sprintf("{{ .Values.%s }}", value)

			for _, container := range pod.Object.Spec.Containers {
				// Skip if the pod already has this container
				if container.Name == value {
					return nil
				}
			}

			pod.Object.Spec.Containers = append(pod.Object.Spec.Containers, v1.Container{
				Name:  value,
				Image: imagePlaceholder,
			})
			if pod.Transformations == nil {
				pod.Transformations = make(map[string]core.IaCValue)
			}
			pod.Transformations[value] = core.IaCValue{ResourceId: image.Id(), Property: resources.ID_IAC_VALUE}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*kubernetes.Deployment, *resources.EcrImage]{
		Configure: func(deployment *kubernetes.Deployment, image *resources.EcrImage, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if deployment.Object == nil {
				return fmt.Errorf("deployment %s has no object", deployment.Name)
			}
			value := resources.GenerateImagePlaceholder(image.Name)
			imagePlaceholder := fmt.Sprintf("{{ .Values.%s }}", value)

			for _, container := range deployment.Object.Spec.Template.Spec.Containers {
				// Skip if the deployment already has this container
				if container.Name == value {
					return nil
				}
			}

			deployment.Object.Spec.Template.Spec.Containers = append(deployment.Object.Spec.Template.Spec.Containers, v1.Container{
				Name:  value,
				Image: imagePlaceholder,
			})
			if deployment.Transformations == nil {
				deployment.Transformations = make(map[string]core.IaCValue)
			}
			deployment.Transformations[value] = core.IaCValue{ResourceId: image.Id(), Property: resources.ID_IAC_VALUE}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*kubernetes.Manifest, *resources.EfsMountTarget]{},
	knowledgebase.EdgeBuilder[*kubernetes.Manifest, *resources.EfsFileSystem]{},
	knowledgebase.EdgeBuilder[*kubernetes.Pod, *resources.EfsMountTarget]{
		Configure: func(pod *kubernetes.Pod, mountTarget *resources.EfsMountTarget, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			_, err := resources.MountEfsVolume(pod, mountTarget, dag, data.AppName)
			return err
		},
	},
	knowledgebase.EdgeBuilder[*kubernetes.Deployment, *resources.EfsMountTarget]{
		Configure: func(deployment *kubernetes.Deployment, mountTarget *resources.EfsMountTarget, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			_, err := resources.MountEfsVolume(deployment, mountTarget, dag, data.AppName)
			return err
		},
	},

	knowledgebase.EdgeBuilder[*kubernetes.HelmChart, *resources.EksFargateProfile]{},
	knowledgebase.EdgeBuilder[*kubernetes.HelmChart, *resources.EksNodeGroup]{},
	knowledgebase.EdgeBuilder[*kubernetes.HelmChart, *resources.EcrImage]{},
	knowledgebase.EdgeBuilder[*kubernetes.HelmChart, *kubernetes.HelmChart]{},
	knowledgebase.EdgeBuilder[*kubernetes.HelmChart, *kubernetes.KustomizeDirectory]{},
	knowledgebase.EdgeBuilder[*kubernetes.HelmChart, *resources.PrivateDnsNamespace]{},
	knowledgebase.EdgeBuilder[*resources.TargetGroup, *kubernetes.TargetGroupBinding]{
		DeploymentOrderReversed: true,
		Reuse:                   knowledgebase.Downstream,
		Configure: func(targetGroup *resources.TargetGroup, tgBinding *kubernetes.TargetGroupBinding, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if tgBinding.Object == nil {
				return fmt.Errorf("%s has no object", tgBinding.Id())
			}
			service, err := core.GetSingleDownstreamResourceOfType[*kubernetes.Service](dag, tgBinding)
			if err != nil {
				return err
			}
			if service.Object == nil {
				return fmt.Errorf("%s has no object", service.Id())
			}
			if service.Object.Name == "" {
				return fmt.Errorf("object in %s has no name", service.Id())
			}
			cluster, ok := core.GetResource[*resources.EksCluster](dag, tgBinding.Cluster)
			if !ok {
				return fmt.Errorf("could not find cluster %s associateed with target binding %s", tgBinding.Cluster, tgBinding.Id())
			}

			// Add the target group ARN to the target group binding
			value := resources.GenerateTargetGroupBindingPlaceholder(targetGroup.Name)
			bindingPlaceholder := fmt.Sprintf("{{ .Values.%s }}", value)
			tgBinding.Object.Spec.TargetGroupARN = bindingPlaceholder

			if tgBinding.Transformations == nil {
				tgBinding.Transformations = make(map[string]core.IaCValue)
			}
			tgBinding.Transformations[value] = core.IaCValue{ResourceId: targetGroup.Id(), Property: resources.ARN_IAC_VALUE}

			// Update the target group binding's service
			tgBinding.Object.Spec.ServiceRef = v1beta1.ServiceReference{
				Name: service.Object.Name,
				Port: intstr.FromInt(int(service.Object.Spec.Ports[0].Port)),
			}

			// Configure the bound target group
			targetGroup.TargetType = "ip"
			targetGroup.Vpc = cluster.Vpc
			// we only support one port per service right now
			targetGroup.Port = int(service.Object.Spec.Ports[0].Port)
			targetGroup.Protocol = string(service.Object.Spec.Ports[0].Protocol)

			// Install the ALB Controller chart on the cluster
			_, err = cluster.InstallAlbController(targetGroup.BaseConstructRefs(), dag, data.AppName)
			return err
		},
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
	knowledgebase.EdgeBuilder[*kubernetes.Pod, *resources.PrivateDnsNamespace]{
		Configure: func(pod *kubernetes.Pod, namespace *resources.PrivateDnsNamespace, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			deploymentRole, err := GetPodServiceAccountRole(pod, dag)
			if err != nil {
				return err
			}
			policy, err := core.CreateResource[*resources.IamPolicy](dag, resources.IamPolicyCreateParams{
				AppName: data.AppName,
				Name:    "servicediscovery",
				Refs:    core.BaseConstructSetOf(pod, namespace),
			})
			if err != nil {
				return err
			}
			dag.AddDependency(policy, namespace)
			dag.AddDependency(deploymentRole, policy)
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*kubernetes.Deployment, *resources.PrivateDnsNamespace]{
		Configure: func(deployment *kubernetes.Deployment, namespace *resources.PrivateDnsNamespace, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			deploymentRole, err := GetDeploymentServiceAccountRole(deployment, dag)
			if err != nil {
				return err
			}
			policy, err := core.CreateResource[*resources.IamPolicy](dag, resources.IamPolicyCreateParams{
				AppName: data.AppName,
				Name:    "servicediscovery",
				Refs:    core.BaseConstructSetOf(deployment, namespace),
			})
			if err != nil {
				return err
			}
			dag.AddDependency(policy, namespace)
			dag.AddDependency(deploymentRole, policy)
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*kubernetes.Deployment, *kubernetes.ServiceExport]{
		Configure: func(deployment *kubernetes.Deployment, serviceExport *kubernetes.ServiceExport, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			exportCluster, ok := core.GetResource[*resources.EksCluster](dag, serviceExport.Cluster)
			if !ok {
				return fmt.Errorf("could not find cluster %s associated with service export %s", serviceExport.Cluster, serviceExport.Id())
			}

			_, err := core.CreateResource[*resources.PrivateDnsNamespace](dag, resources.PrivateDnsNamespaceCreateParams{
				Refs:    core.BaseConstructSetOf(serviceExport, deployment),
				AppName: data.AppName,
			})
			if err != nil {
				return err
			}

			cmController, err := exportCluster.InstallCloudMapController(core.BaseConstructSetOf(serviceExport, deployment), dag)
			dag.AddDependency(serviceExport, cmController)
			return err
		},
	},
	knowledgebase.EdgeBuilder[*kubernetes.Pod, *kubernetes.ServiceExport]{
		Configure: func(pod *kubernetes.Pod, serviceExport *kubernetes.ServiceExport, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			exportCluster, ok := core.GetResource[*resources.EksCluster](dag, serviceExport.Cluster)
			if !ok {
				return fmt.Errorf("could not find cluster %s associated with service export %s", serviceExport.Cluster, serviceExport.Id())
			}

			_, err := core.CreateResource[*resources.PrivateDnsNamespace](dag, resources.PrivateDnsNamespaceCreateParams{
				Refs:    core.BaseConstructSetOf(serviceExport, pod),
				AppName: data.AppName,
			})
			if err != nil {
				return err
			}

			cmController, err := exportCluster.InstallCloudMapController(core.BaseConstructSetOf(serviceExport, pod), dag)
			dag.AddDependency(serviceExport, cmController)
			return err
		},
	},

	knowledgebase.EdgeBuilder[*resources.PrivateDnsNamespace, *kubernetes.ServiceExport]{},
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
		return nil, fmt.Errorf("no service account found for deployment %s during expansion", deployment.Id())
	}
	role, err := resources.GetServiceAccountRole(sa, dag)
	if err != nil {
		return nil, err
	}
	return role, nil
}
