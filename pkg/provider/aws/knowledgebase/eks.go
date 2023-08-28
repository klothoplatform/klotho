package knowledgebase

import (
	"fmt"
	"strings"

	k8sSanitizer "github.com/klothoplatform/klotho/pkg/sanitization/kubernetes"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/aws-load-balancer-controller/apis/elbv2/v1beta1"

	"github.com/klothoplatform/klotho/pkg/construct"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	kubernetes "github.com/klothoplatform/klotho/pkg/provider/kubernetes/resources"
)

var EksKB = knowledgebase.Build(
	knowledgebase.EdgeBuilder[*resources.EksFargateProfile, *resources.EksCluster]{
		Configure: func(profile *resources.EksFargateProfile, cluster *resources.EksCluster, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
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
		Configure: func(nodeGroup *resources.EksNodeGroup, cluster *resources.EksCluster, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
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
		Configure: func(sa *kubernetes.ServiceAccount, role *resources.IamRole, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
			if sa.Object == nil {
				return fmt.Errorf("%s has no object", sa.Id())
			}
			if sa.Cluster.IsZero() {
				return fmt.Errorf("%s has no cluster", sa.Id())
			}
			value := resources.GenerateRoleArnPlaceholder(role.Name)
			roleArnPlaceholder := fmt.Sprintf("{{ .Values.%s }}", value)

			if sa.Object.Annotations == nil {
				sa.Object.Annotations = make(map[string]string)
			}
			sa.Object.Annotations["eks.amazonaws.com/role-arn"] = roleArnPlaceholder

			if sa.Values == nil {
				sa.Values = make(map[string]construct.IaCValue)
			}
			sa.Values[value] = construct.IaCValue{ResourceId: role.Id(), Property: resources.ARN_IAC_VALUE}

			// Sets the role's AssumeRolePolicyDocument to allow the service account to assume the role
			oidc, err := construct.CreateResource[*resources.OpenIdConnectProvider](dag, resources.OidcCreateParams{
				AppName:     data.AppName,
				ClusterName: sa.Cluster.Name,
				Refs:        construct.BaseConstructSetOf(sa),
			})
			if err != nil {
				return err
			}
			assumeRolePolicy := resources.GetServiceAccountAssumeRolePolicy(sa.Object.Name, sa.Object.Namespace, oidc)
			role.AssumeRolePolicyDoc = assumeRolePolicy
			dag.AddDependencyWithData(role, oidc, data)
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.EksAddon, *resources.EksCluster]{},
	knowledgebase.EdgeBuilder[*resources.EksAddon, *resources.IamRole]{},
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
		Configure: func(pod *kubernetes.Pod, image *resources.EcrImage, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {

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
				Name:  k8sSanitizer.RFC1123LabelSanitizer.Apply(value),
				Image: imagePlaceholder,
			})
			if pod.Values == nil {
				pod.Values = make(map[string]construct.IaCValue)
			}
			pod.Values[value] = construct.IaCValue{ResourceId: image.Id(), Property: resources.ID_IAC_VALUE}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*kubernetes.Deployment, *resources.EcrImage]{
		Configure: func(deployment *kubernetes.Deployment, image *resources.EcrImage, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
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
			if deployment.Values == nil {
				deployment.Values = make(map[string]construct.IaCValue)
			}
			deployment.Values[value] = construct.IaCValue{ResourceId: image.Id(), Property: resources.ID_IAC_VALUE}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*kubernetes.Pod, *resources.EfsMountTarget]{},
	knowledgebase.EdgeBuilder[*kubernetes.Pod, *resources.EfsFileSystem]{
		Configure: func(pod *kubernetes.Pod, fileSystem *resources.EfsFileSystem, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
			return mountEfsFileSystemToPodOrDeployment(pod, fileSystem, dag, data)
		},
	},
	knowledgebase.EdgeBuilder[*kubernetes.Deployment, *resources.EfsMountTarget]{},
	knowledgebase.EdgeBuilder[*kubernetes.PersistentVolume, *resources.EfsFileSystem]{},
	knowledgebase.EdgeBuilder[*kubernetes.Deployment, *resources.EfsFileSystem]{
		Configure: func(deployment *kubernetes.Deployment, fileSystem *resources.EfsFileSystem, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
			return mountEfsFileSystemToPodOrDeployment(deployment, fileSystem, dag, data)
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
		Configure: func(targetGroup *resources.TargetGroup, tgBinding *kubernetes.TargetGroupBinding, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
			if tgBinding.Object == nil {
				return fmt.Errorf("%s has no object", tgBinding.Id())
			}
			service, err := construct.GetSingleDownstreamResourceOfType[*kubernetes.Service](dag, tgBinding)
			if err != nil {
				return err
			}
			if service.Object == nil {
				return fmt.Errorf("%s has no object", service.Id())
			}
			if service.Object.Name == "" {
				return fmt.Errorf("object in %s has no name", service.Id())
			}
			cluster, ok := construct.GetResource[*resources.EksCluster](dag, tgBinding.Cluster)
			if !ok {
				return fmt.Errorf("could not find cluster %s associateed with target binding %s", tgBinding.Cluster, tgBinding.Id())
			}

			// Add the target group ARN to the target group binding
			value := resources.GenerateTargetGroupBindingPlaceholder(targetGroup.Name)
			bindingPlaceholder := fmt.Sprintf("{{ .Values.%s }}", value)
			tgBinding.Object.Spec.TargetGroupARN = bindingPlaceholder

			if tgBinding.Values == nil {
				tgBinding.Values = make(map[string]construct.IaCValue)
			}
			tgBinding.Values[value] = construct.IaCValue{ResourceId: targetGroup.Id(), Property: resources.ARN_IAC_VALUE}

			if len(service.Object.Spec.Ports) == 0 {
				return fmt.Errorf("service %s has no ports", service.Id())
			}
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
		Configure: func(pod *kubernetes.Pod, namespace *resources.PrivateDnsNamespace, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
			deploymentRole, err := GetPodServiceAccountRole(pod, dag)
			if err != nil {
				return err
			}
			policy, err := construct.CreateResource[*resources.IamPolicy](dag, resources.IamPolicyCreateParams{
				AppName: data.AppName,
				Name:    "servicediscovery",
				Refs:    construct.BaseConstructSetOf(pod, namespace),
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
		Configure: func(deployment *kubernetes.Deployment, namespace *resources.PrivateDnsNamespace, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
			deploymentRole, err := GetDeploymentServiceAccountRole(deployment, dag)
			if err != nil {
				return err
			}
			policy, err := construct.CreateResource[*resources.IamPolicy](dag, resources.IamPolicyCreateParams{
				AppName: data.AppName,
				Name:    "servicediscovery",
				Refs:    construct.BaseConstructSetOf(deployment, namespace),
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
		Configure: func(deployment *kubernetes.Deployment, serviceExport *kubernetes.ServiceExport, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
			exportCluster, ok := construct.GetResource[*resources.EksCluster](dag, serviceExport.Cluster)
			if !ok {
				return fmt.Errorf("could not find cluster %s associated with service export %s", serviceExport.Cluster, serviceExport.Id())
			}

			_, err := construct.CreateResource[*resources.PrivateDnsNamespace](dag, resources.PrivateDnsNamespaceCreateParams{
				Refs:    construct.BaseConstructSetOf(serviceExport, deployment),
				AppName: data.AppName,
			})
			if err != nil {
				return err
			}

			cmController, err := exportCluster.InstallCloudMapController(construct.BaseConstructSetOf(serviceExport, deployment), dag)
			dag.AddDependency(serviceExport, cmController)
			return err
		},
	},
	knowledgebase.EdgeBuilder[*kubernetes.Pod, *kubernetes.ServiceExport]{
		Configure: func(pod *kubernetes.Pod, serviceExport *kubernetes.ServiceExport, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
			exportCluster, ok := construct.GetResource[*resources.EksCluster](dag, serviceExport.Cluster)
			if !ok {
				return fmt.Errorf("could not find cluster %s associated with service export %s", serviceExport.Cluster, serviceExport.Id())
			}

			_, err := construct.CreateResource[*resources.PrivateDnsNamespace](dag, resources.PrivateDnsNamespaceCreateParams{
				Refs:    construct.BaseConstructSetOf(serviceExport, pod),
				AppName: data.AppName,
			})
			if err != nil {
				return err
			}

			cmController, err := exportCluster.InstallCloudMapController(construct.BaseConstructSetOf(serviceExport, pod), dag)
			dag.AddDependency(serviceExport, cmController)
			return err
		},
	},

	knowledgebase.EdgeBuilder[*resources.PrivateDnsNamespace, *kubernetes.ServiceExport]{},
	knowledgebase.EdgeBuilder[*kubernetes.Pod, *resources.DynamodbTable]{
		Configure: func(pod *kubernetes.Pod, table *resources.DynamodbTable, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
			role, err := GetPodServiceAccountRole(pod, dag)
			if err != nil {
				return err
			}
			dag.AddDependency(role, table)
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*kubernetes.Deployment, *resources.DynamodbTable]{
		Configure: func(deployment *kubernetes.Deployment, table *resources.DynamodbTable, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
			role, err := GetDeploymentServiceAccountRole(deployment, dag)
			if err != nil {
				return err
			}
			dag.AddDependency(role, table)
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*kubernetes.Pod, *resources.ElasticacheCluster]{
		Configure: func(pod *kubernetes.Pod, cluster *resources.ElasticacheCluster, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*kubernetes.Deployment, *resources.ElasticacheCluster]{
		Configure: func(deployment *kubernetes.Deployment, cluster *resources.ElasticacheCluster, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*kubernetes.Pod, *resources.S3Bucket]{
		Configure: func(pod *kubernetes.Pod, bucket *resources.S3Bucket, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
			role, err := GetPodServiceAccountRole(pod, dag)
			if err != nil {
				return err
			}
			dag.AddDependency(role, bucket)
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*kubernetes.Deployment, *resources.S3Bucket]{
		Configure: func(deployment *kubernetes.Deployment, bucket *resources.S3Bucket, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
			role, err := GetDeploymentServiceAccountRole(deployment, dag)
			if err != nil {
				return err
			}
			dag.AddDependency(role, bucket)
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*kubernetes.Pod, *resources.RdsInstance]{
		Configure: func(pod *kubernetes.Pod, instance *resources.RdsInstance, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
			role, err := GetPodServiceAccountRole(pod, dag)
			if err != nil {
				return err
			}
			dag.AddDependency(role, instance)
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*kubernetes.Deployment, *resources.RdsInstance]{
		Configure: func(deployment *kubernetes.Deployment, instance *resources.RdsInstance, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
			role, err := GetDeploymentServiceAccountRole(deployment, dag)
			if err != nil {
				return err
			}
			dag.AddDependency(role, instance)
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*kubernetes.Pod, *resources.RdsProxy]{
		Configure: func(pod *kubernetes.Pod, proxy *resources.RdsProxy, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
			role, err := GetPodServiceAccountRole(pod, dag)
			if err != nil {
				return err
			}
			dag.AddDependency(role, proxy)
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*kubernetes.Deployment, *resources.RdsProxy]{
		Configure: func(deployment *kubernetes.Deployment, proxy *resources.RdsProxy, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
			role, err := GetDeploymentServiceAccountRole(deployment, dag)
			if err != nil {
				return err
			}
			dag.AddDependency(role, proxy)
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*kubernetes.PersistentVolume, *resources.EksCluster]{},
	knowledgebase.EdgeBuilder[*kubernetes.PersistentVolumeClaim, *resources.EksCluster]{},
	knowledgebase.EdgeBuilder[*kubernetes.StorageClass, *resources.EksCluster]{},
)

func mountEfsFileSystemToPodOrDeployment(computeResource construct.Resource, fileSystem *resources.EfsFileSystem, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
	switch computeResource.(type) {
	case *kubernetes.Pod, *kubernetes.Deployment:
	default:
		return fmt.Errorf("cannot mount EFS filesystem to %s", computeResource.Id())
	}

	if computeResource == nil {
		return fmt.Errorf("%s has no object", computeResource.Id())
	}

	// Ensure that the filesystem has a mount target in the same AZs as the computeResource's pod(s)
	deploymentSubnets := getSubnetsForPodOrDeployment(dag, computeResource)
	if len(deploymentSubnets) == 0 {
		return fmt.Errorf("%s is not associated with any subnets", computeResource.Id())
	}

	existingMountTargets := construct.GetUpstreamResourcesOfType[*resources.EfsMountTarget](dag, fileSystem)

	var mountTargetAZs = make(map[string]bool)
	for _, mountTarget := range existingMountTargets {
		if mountTarget.Subnet == nil {
			return fmt.Errorf("%s has no subnet", mountTarget.Id())
		}
		if mountTarget.Subnet.AvailabilityZone.ResourceId.IsZero() {
			return fmt.Errorf("%s has no AZ", mountTarget.Subnet.Id())
		}
		mountTargetAZs[mountTarget.Subnet.AvailabilityZone.Property] = true
	}

	// Create mount targets for any AZs that don't already have one
	for _, subnet := range deploymentSubnets {
		if subnet.AvailabilityZone.ResourceId.IsZero() {
			continue
		}
		if _, ok := mountTargetAZs[subnet.AvailabilityZone.Property]; !ok {
			mountTarget, err := construct.CreateResource[*resources.EfsMountTarget](dag, resources.EfsMountTargetCreateParams{
				Name:          fmt.Sprintf("%s-%s", fileSystem.Name, subnet.Name),
				ConstructRefs: construct.BaseConstructSetOf(computeResource, fileSystem),
			})
			if err != nil {
				return err
			}
			mountTarget.Subnet = subnet
			mountTargetAZs[subnet.AvailabilityZone.Property] = true
			dag.AddDependencyWithData(computeResource, mountTarget, data)
			dag.AddDependency(mountTarget, subnet)
		}
	}

	_, err := resources.CreatePersistentVolume(computeResource, fileSystem, dag, data.AppName)
	return err
}

func getSubnetsForPodOrDeployment(dag *construct.ResourceGraph, resource construct.Resource) []*resources.Subnet {
	var subnets []*resources.Subnet

	var deploymentTargets []construct.Resource

	for _, downstream := range dag.GetDownstreamResources(resource) {
		switch downstream.(type) {
		case *resources.EksNodeGroup, *resources.EksFargateProfile:
			deploymentTargets = append(deploymentTargets, downstream)
		}
	}

	for _, deploymentTarget := range deploymentTargets {
		subnets = append(subnets, construct.GetDownstreamResourcesOfType[*resources.Subnet](dag, deploymentTarget)...)
	}

	return subnets
}

func GetPodServiceAccountRole(pod *kubernetes.Pod, dag *construct.ResourceGraph) (*resources.IamRole, error) {
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

func GetDeploymentServiceAccountRole(deployment *kubernetes.Deployment, dag *construct.ResourceGraph) (*resources.IamRole, error) {
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
