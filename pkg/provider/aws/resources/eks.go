package resources

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"reflect"
	"strings"

	"github.com/klothoplatform/klotho/pkg/core"
	kubernetes "github.com/klothoplatform/klotho/pkg/provider/kubernetes/resources"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
	k8sSanitizer "github.com/klothoplatform/klotho/pkg/sanitization/kubernetes"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	EKS_CLUSTER_TYPE         = "eks_cluster"
	EKS_FARGATE_PROFILE_TYPE = "eks_fargate_profile"
	EKS_NODE_GROUP_TYPE      = "eks_node_group"
	DEFAULT_CLUSTER_NAME     = "eks-cluster"
	EKS_ADDON_TYPE           = "eks_addon"

	OIDC_SUB_IAC_VALUE                            = "oidc_url"
	OIDC_AUD_IAC_VALUE                            = "oidc_aud"
	CLUSTER_ENDPOINT_IAC_VALUE                    = "cluster_endpoint"
	CLUSTER_CA_DATA_IAC_VALUE                     = "cluster_certificate_authority_data"
	CLUSTER_PROVIDER_IAC_VALUE                    = "cluster_provider"
	CLUSTER_SECURITY_GROUP_ID_IAC_VALUE           = "cluster_security_group_id"
	NAME_IAC_VALUE                                = "name"
	ID_IAC_VALUE                                  = "id"
	AWS_OBSERVABILITY_CONFIG_MAP_REGION_IAC_VALUE = "aws_observ_cm_region"
	NODE_GROUP_NAME_IAC_VALUE                     = "node_group_name"

	AWS_OBSERVABILITY_NS_PATH         = "aws_observability_namespace.yaml"
	AWS_OBSERVABILITY_CONFIG_MAP_PATH = "aws_observability_configmap.yaml"
	AMAZON_CLOUDWATCH_NS_PATH         = "amazon_cloudwatch_namespace.yaml"
	FLUENT_BIT_CLUSTER_INFO           = "fluent_bit_cluster_info.yaml"
	CM_CLUSTER_SET                    = "cloudmap_cluster_set.yaml"
	MANIFEST_PATH_PREFIX              = "manifests"
)

var nodeGroupSanitizer = aws.EksNodeGroupSanitizer
var profileSanitizer = aws.EksFargateProfileSanitizer
var clusterSanitizer = aws.EksClusterSanitizer

var EKS_AMI_INSTANCE_PREFIX_MAP = map[string][]string{
	"AL2_x86_64": {
		"c1",
		"c3",
		"c4",
		"c5a",
		"c5d",
		"c5n",
		"c6i",
		"d2",
		"i2",
		"i3",
		"i3en",
		"i4i",
		"inf1",
		"m1",
		"m2",
		"m3",
		"m4",
		"m5",
		"m5a",
		"m5ad",
		"m5d",
		"m5zn",
		"m6i",
		"r3",
		"r4",
		"r5",
		"r5a",
		"r5ad",
		"r5d",
		"r5n",
		"r6i",
		"t1",
		"t2",
		"t3",
		"t3a",
		"z1d",
	},
	"AL2_x86_64_GPU": {"g2", "g3", "g4dn"},
	"AL2_ARM_64":     {"c6g", "c6gd", "c6gn", "m6g", "m6gd", "r6g", "r6gd", "t4g"},
}

//go:embed manifests/*
var eksManifests embed.FS

type (
	EksCluster struct {
		Name           string
		ConstructsRef  core.BaseConstructSet `yaml:"-"`
		ClusterRole    *IamRole
		Vpc            *Vpc
		Subnets        []*Subnet
		SecurityGroups []*SecurityGroup
		Kubeconfig     *kubernetes.Kubeconfig `yaml:"-"`
	}

	EksFargateProfile struct {
		Name             string
		ConstructsRef    core.BaseConstructSet `yaml:"-"`
		Cluster          *EksCluster
		PodExecutionRole *IamRole
		Selectors        []*FargateProfileSelector
		Subnets          []*Subnet
	}

	FargateProfileSelector struct {
		Namespace string
		Labels    map[string]string
	}

	EksNodeGroup struct {
		Name           string
		ConstructsRef  core.BaseConstructSet `yaml:"-"`
		Cluster        *EksCluster
		NodeRole       *IamRole
		AmiType        string
		Subnets        []*Subnet
		DesiredSize    int
		MinSize        int
		MaxSize        int
		MaxUnavailable int
		DiskSize       int
		InstanceTypes  []string
		Labels         map[string]string
	}

	EksAddon struct {
		Name          string
		ConstructsRef core.BaseConstructSet `yaml:"-"`
		AddonName     string
		ClusterName   *AwsResourceValue
	}
)

type EksClusterCreateParams struct {
	Refs    core.BaseConstructSet
	AppName string
	Name    string
}

func (cluster *EksCluster) Create(dag *core.ResourceGraph, params EksClusterCreateParams) error {

	cluster.Name = clusterSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	cluster.ConstructsRef = params.Refs.Clone()
	existingCluster := dag.GetResource(cluster.Id())
	if existingCluster != nil {
		graphCluster := existingCluster.(*EksCluster)
		graphCluster.ConstructsRef.AddAll(params.Refs)
	} else {
		dag.AddResource(cluster)

		// We create these add ons in cluster creation since there is edge which would create them
		// These are always installed in every cluster, no matter the configuration
		cluster.installVpcCniAddon(cluster.ConstructsRef, dag)
	}
	return nil
}

func (cluster *EksCluster) MakeOperational(dag *core.ResourceGraph, appName string) error {
	zap.S().Debugf("Making cluster %s operational", cluster.Name)
	if cluster.ClusterRole == nil {
		roles := core.GetDownstreamResourcesOfType[*IamRole](dag, cluster)
		if len(roles) > 1 {
			return errors.Errorf("cluster %s has multiple roles", cluster.Name)
		} else if len(roles) == 0 {
			err := dag.CreateDependencies(cluster, map[string]any{
				"ClusterRole": RoleCreateParams{
					AppName: appName,
					Name:    fmt.Sprintf("%s-ClusterAdmin", cluster.Name),
					Refs:    core.BaseConstructSetOf(cluster),
				},
			})
			if err != nil {
				return err
			}
		} else {
			cluster.ClusterRole = roles[0]
		}
	}

	if cluster.Vpc == nil {
		vpc, err := getSingleUpstreamVpc(dag, cluster)
		if err != nil {
			return err
		}
		if vpc != nil {
			cluster.Vpc = vpc
			dag.AddDependency(cluster, vpc)
		}
	}

	// We want to add this to ensure that if the vpc is set it has an edge in the graph so the sgs and subnets are checked against it
	dag.AddDependenciesReflect(cluster)

	if len(cluster.Subnets) == 0 {
		subnets, err := getSubnetsOperational(dag, cluster, appName)
		if err != nil {
			return err
		}
		cluster.Subnets = subnets
		dag.AddDependenciesReflect(cluster)
	}

	if len(cluster.SecurityGroups) == 0 {
		sgs, err := getSecurityGroupsOperational(dag, cluster, appName)
		if err != nil {
			return err
		}
		cluster.SecurityGroups = append(cluster.SecurityGroups, sgs...)
		dag.AddDependenciesReflect(cluster)
	}

	if cluster.Vpc == nil {
		vpc, err := getSingleUpstreamVpc(dag, cluster)
		if err != nil {
			return err
		}
		if vpc == nil {
			return fmt.Errorf("cluster %s has no vpc", cluster.Name)
		}
		cluster.Vpc = vpc
	}

	dag.AddDependenciesReflect(cluster)
	return nil
}

type EksClusterConfigureParams struct {
}

func (cluster *EksCluster) Configure(params EksClusterConfigureParams) error {
	// Add the kubeconfig after the dependencies are added otherwise we will have a circular dependency
	cluster.Kubeconfig = createEKSKubeconfig(cluster, NewRegion())
	return nil
}

type EksFargateProfileCreateParams struct {
	Refs    core.BaseConstructSet
	AppName string
	Name    string
}

func (profile *EksFargateProfile) Create(dag *core.ResourceGraph, params EksFargateProfileCreateParams) error {
	profile.Name = profileSanitizer.Apply(fmt.Sprintf("%s_%s", params.AppName, params.Name))

	existingProfile, found := core.GetResource[*EksFargateProfile](dag, profile.Id())
	if found {
		existingProfile.ConstructsRef.AddAll(params.Refs)
	} else {
		profile.ConstructsRef = params.Refs.Clone()
		dag.AddResource(profile)
	}
	return nil
}

func (profile *EksFargateProfile) MakeOperational(dag *core.ResourceGraph, appName string) error {

	if profile.Cluster == nil {
		clusters := core.GetDownstreamResourcesOfType[*EksCluster](dag, profile)
		if len(clusters) > 1 {
			return fmt.Errorf("fargate profile %s has multiple clusters", profile.Id())
		} else if len(clusters) == 0 {
			err := dag.CreateDependencies(profile, map[string]any{
				"Cluster": EksClusterCreateParams{
					AppName: appName,
					Name:    DEFAULT_CLUSTER_NAME,
					Refs:    core.BaseConstructSetOf(profile),
				},
			})
			if err != nil {
				return err
			}
		} else {
			profile.Cluster = clusters[0]
		}
	}

	if len(profile.Subnets) == 0 {
		subnets, err := getSubnetsOperational(dag, profile, appName)
		if err != nil {
			return err
		}
		for _, subnet := range subnets {
			if subnet.Type == PrivateSubnet {
				profile.Subnets = append(profile.Subnets, subnet)
			}
		}
	}

	if profile.PodExecutionRole == nil {
		roles := core.GetDownstreamResourcesOfType[*IamRole](dag, profile)
		if len(roles) > 1 {
			return fmt.Errorf("fargate profile %s has multiple roles", profile.Id())
		} else if len(roles) == 0 {
			err := dag.CreateDependencies(profile, map[string]any{
				"PodExecutionRole": RoleCreateParams{
					AppName: appName,
					Name:    fmt.Sprintf("%s-PodExecutionRole", profile.Name),
					Refs:    core.BaseConstructSetOf(profile),
				},
			})
			if err != nil {
				return err
			}
		} else {
			profile.PodExecutionRole = roles[0]
		}
	}

	dag.AddDependenciesReflect(profile)
	return nil
}

type EksFargateProfileConfigureParams struct {
	Namespace string
}

func (profile *EksFargateProfile) Configure(params EksFargateProfileConfigureParams) error {
	namespace := "default"
	if params.Namespace != "" {
		namespace = params.Namespace
	}
	addSelector := true
	for _, selector := range profile.Selectors {
		if selector.Namespace == namespace {
			addSelector = false
		}
	}
	if addSelector {
		profile.Selectors = append(profile.Selectors, &FargateProfileSelector{Namespace: namespace, Labels: map[string]string{"klotho-fargate-enabled": "true"}})
	}
	return nil
}

type EksNodeGroupCreateParams struct {
	InstanceType string
	NetworkType  string
	Refs         core.BaseConstructSet
	AppName      string
}

func (nodeGroup *EksNodeGroup) Create(dag *core.ResourceGraph, params EksNodeGroupCreateParams) error {

	name := NodeGroupName(params.NetworkType, params.InstanceType)
	nodeGroup.Name = fmt.Sprintf("%s_%s", params.AppName, name)
	existingNodeGroup, found := core.GetResource[*EksNodeGroup](dag, nodeGroup.Id())
	if found {
		existingNodeGroup.ConstructsRef.AddAll(params.Refs)
	} else {
		nodeGroup.ConstructsRef = params.Refs.Clone()
		nodeGroup.InstanceTypes = []string{params.InstanceType}
		nodeGroup.Labels = map[string]string{
			"network_placement": params.NetworkType,
		}
		dag.AddResource(nodeGroup)
	}

	return nil
}

func (nodeGroup *EksNodeGroup) MakeOperational(dag *core.ResourceGraph, appName string) error {
	if nodeGroup.Cluster == nil {
		copyNodeGroup := *nodeGroup
		clusters := core.GetDownstreamResourcesOfType[*EksCluster](dag, nodeGroup)
		if len(clusters) > 1 {
			return fmt.Errorf("node group %s has multiple clusters", nodeGroup.Id())
		} else if len(clusters) == 0 {
			err := dag.CreateDependencies(nodeGroup, map[string]any{
				"Cluster": EksClusterCreateParams{
					AppName: appName,
					Name:    DEFAULT_CLUSTER_NAME,
					Refs:    core.BaseConstructSetOf(nodeGroup),
				},
			})
			if err != nil {
				return err
			}
		} else {
			nodeGroup.Cluster = clusters[0]
		}
		// We want to add the cluster name to the node group name to prevent conflicts and then do an in place replacement in the graph
		nodeGroup.Name = fmt.Sprintf("%s-%s", nodeGroup.Cluster.Name, nodeGroup.Name)
		err := dag.ReplaceConstruct(&copyNodeGroup, nodeGroup)
		if err != nil {
			return err
		}
		dag.AddDependenciesReflect(nodeGroup)
	}

	if nodeGroup.NodeRole == nil {
		roles := core.GetDownstreamResourcesOfType[*IamRole](dag, nodeGroup)
		if len(roles) > 1 {
			return fmt.Errorf("node group %s has multiple roles", nodeGroup.Id())
		} else if len(roles) == 0 {
			err := dag.CreateDependencies(nodeGroup, map[string]any{
				"NodeRole": RoleCreateParams{
					AppName: appName,
					Name:    fmt.Sprintf("%s-NodeRole", nodeGroup.Name),
					Refs:    core.BaseConstructSetOf(nodeGroup),
				},
			})
			if err != nil {
				return err
			}
		} else {
			nodeGroup.NodeRole = roles[0]
		}
	}

	if len(nodeGroup.Subnets) == 0 {
		subnets, err := getSubnetsOperational(dag, nodeGroup, appName)
		if err != nil {
			return err
		}
		networkPlacement := nodeGroup.Labels["network_placement"]
		if networkPlacement == "" {
			networkPlacement = PrivateSubnet
		}
		for _, subnet := range subnets {
			if subnet.Type == networkPlacement {
				nodeGroup.Subnets = append(nodeGroup.Subnets, subnet)
			}
		}
	}

	dag.AddDependenciesReflect(nodeGroup)
	return nil
}

type EksNodeGroupConfigureParams struct {
	DiskSize int
}

func (nodeGroup *EksNodeGroup) Configure(params EksNodeGroupConfigureParams) error {
	nodeGroup.AmiType = amiFromInstanceType(nodeGroup.InstanceTypes[0])
	nodeGroup.DesiredSize = 2
	nodeGroup.MaxSize = 2
	nodeGroup.MinSize = 1
	nodeGroup.MaxUnavailable = 1
	nodeGroup.DiskSize = params.DiskSize
	return nil
}

func (cluster *EksCluster) SetUpDefaultNodeGroup(dag *core.ResourceGraph, appName string) error {
	_, err := core.CreateResource[*EksNodeGroup](dag, EksNodeGroupCreateParams{
		InstanceType: "t3.medium",
		NetworkType:  PrivateSubnet,
		Refs:         cluster.ConstructsRef,
		AppName:      appName,
	})
	if err != nil {
		return err
	}
	cluster.CreatePrerequisiteCharts(dag)
	err = cluster.InstallFluentBit(cluster.ConstructsRef, dag)
	if err != nil {
		return err
	}
	return nil
}

func NodeGroupName(networkPlacement string, instanceType string) string {
	return nodeGroupSanitizer.Apply(fmt.Sprintf("%s_%s", networkPlacement, instanceType))
}

func (cluster *EksCluster) CreatePrerequisiteCharts(dag *core.ResourceGraph) {
	charts := []*kubernetes.HelmChart{
		{
			Name:          cluster.Name + `-metrics-server`,
			Chart:         "metrics-server",
			ConstructRefs: cluster.ConstructsRef,
			Cluster: &AwsResourceValue{
				ResourceVal: cluster,
				PropertyVal: CLUSTER_PROVIDER_IAC_VALUE,
			},
			Repo: `https://kubernetes-sigs.github.io/metrics-server/`,
		},
		{
			Name:          cluster.Name + `-cert-manager`,
			Chart:         `cert-manager`,
			ConstructRefs: cluster.ConstructsRef,
			Cluster: &AwsResourceValue{
				ResourceVal: cluster,
				PropertyVal: CLUSTER_PROVIDER_IAC_VALUE,
			},
			Repo:    `https://charts.jetstack.io`,
			Version: `v1.10.0`,
			Values: map[string]any{
				`installCRDs`: true,
				`webhook`: map[string]any{
					`timeoutSeconds`: 30,
				},
			},
		},
	}
	for _, chart := range charts {
		for _, nodeGroup := range cluster.GetClustersNodeGroups(dag) {
			dag.AddDependency(chart, nodeGroup)
		}
	}
}

func (cluster *EksCluster) InstallNvidiaDevicePlugin(dag *core.ResourceGraph) {
	manifest := &kubernetes.Manifest{
		Name:     fmt.Sprintf("%s-%s", cluster.Name, "nvidia-device-plugin"),
		FilePath: "https://raw.githubusercontent.com/NVIDIA/k8s-device-plugin/v1.10/nvidia-device-plugin.yml",
		Cluster: &AwsResourceValue{
			ResourceVal: cluster,
			PropertyVal: CLUSTER_PROVIDER_IAC_VALUE,
		},
	}
	dag.AddDependenciesReflect(manifest)

	for _, ng := range cluster.GetClustersNodeGroups(dag) {
		dag.AddDependency(manifest, ng)
		if strings.HasSuffix(strings.ToLower(ng.AmiType), "_gpu") {
			manifest.ConstructRefs.AddAll(ng.ConstructsRef)
		}
	}
}

func (cluster *EksCluster) CreateFargateLogging(references core.BaseConstructSet, dag *core.ResourceGraph) error {
	namespaceOutputPath := path.Join(MANIFEST_PATH_PREFIX, AWS_OBSERVABILITY_NS_PATH)
	content, err := fs.ReadFile(eksManifests, namespaceOutputPath)
	if err != nil {
		return err
	}
	namespace := &kubernetes.Manifest{
		Name:          fmt.Sprintf("%s-%s", cluster.Name, "aws-observability-ns"),
		ConstructRefs: references,
		FilePath:      namespaceOutputPath,
		Content:       content,
		Cluster: &AwsResourceValue{
			ResourceVal: cluster,
			PropertyVal: CLUSTER_PROVIDER_IAC_VALUE,
		},
	}
	dag.AddResource(namespace)
	dag.AddDependency(namespace, cluster)

	configMapOutputPath := path.Join(MANIFEST_PATH_PREFIX, AWS_OBSERVABILITY_CONFIG_MAP_PATH)
	content, err = fs.ReadFile(eksManifests, configMapOutputPath)
	if err != nil {
		return err
	}
	configMap := &kubernetes.Manifest{
		Name:          fmt.Sprintf("%s-%s", cluster.Name, "aws-observability-config-map"),
		ConstructRefs: references,
		FilePath:      configMapOutputPath,
		Content:       content,
		Cluster: &AwsResourceValue{
			ResourceVal: cluster,
			PropertyVal: CLUSTER_PROVIDER_IAC_VALUE,
		},
		Transformations: map[string]core.IaCValue{
			`data["output.conf"]`: &AwsResourceValue{ResourceVal: cluster, PropertyVal: AWS_OBSERVABILITY_CONFIG_MAP_REGION_IAC_VALUE},
		},
	}
	dag.AddDependenciesReflect(configMap)
	dag.AddDependency(configMap, NewRegion())
	dag.AddDependency(configMap, namespace)
	return nil
}

func (cluster *EksCluster) InstallFluentBit(references core.BaseConstructSet, dag *core.ResourceGraph) error {
	namespaceOutputPath := path.Join(MANIFEST_PATH_PREFIX, AMAZON_CLOUDWATCH_NS_PATH)
	content, err := fs.ReadFile(eksManifests, namespaceOutputPath)
	if err != nil {
		return err
	}
	namespace := &kubernetes.Manifest{
		Name:          fmt.Sprintf("%s-%s", cluster.Name, "awmazon-cloudwatch-ns"),
		ConstructRefs: references,
		FilePath:      namespaceOutputPath,
		Content:       content,
		Cluster: &AwsResourceValue{
			ResourceVal: cluster,
			PropertyVal: CLUSTER_PROVIDER_IAC_VALUE,
		},
	}
	dag.AddResource(namespace)
	dag.AddDependency(namespace, cluster)

	configMapOutputPath := path.Join(MANIFEST_PATH_PREFIX, FLUENT_BIT_CLUSTER_INFO)
	content, err = fs.ReadFile(eksManifests, configMapOutputPath)
	if err != nil {
		return err
	}
	region := NewRegion()
	configMap := &kubernetes.Manifest{
		Name:          fmt.Sprintf("%s-%s", cluster.Name, "fluent-bit-cluster-info-config-map"),
		ConstructRefs: references,
		FilePath:      configMapOutputPath,
		Content:       content,
		Transformations: map[string]core.IaCValue{
			`data["cluster.name"]`: &AwsResourceValue{ResourceVal: cluster, PropertyVal: NAME_IAC_VALUE},
			`data["logs.region"]`:  &AwsResourceValue{ResourceVal: region, PropertyVal: NAME_IAC_VALUE},
		},
		Cluster: &AwsResourceValue{
			ResourceVal: cluster,
			PropertyVal: CLUSTER_PROVIDER_IAC_VALUE,
		},
	}
	dag.AddResource(configMap)
	dag.AddDependency(configMap, cluster)
	dag.AddDependency(configMap, namespace)
	fluentBitOptimized := &kubernetes.Manifest{
		Name:          fmt.Sprintf("%s-%s", cluster.Name, "fluent-bit"),
		ConstructRefs: references,
		FilePath:      "https://raw.githubusercontent.com/aws-samples/amazon-cloudwatch-container-insights/latest/k8s-deployment-manifest-templates/deployment-mode/daemonset/container-insights-monitoring/fluent-bit/fluent-bit.yaml",
		Cluster: &AwsResourceValue{
			ResourceVal: cluster,
			PropertyVal: CLUSTER_PROVIDER_IAC_VALUE,
		},
	}
	dag.AddResource(configMap)
	dag.AddDependency(fluentBitOptimized, cluster)
	dag.AddDependency(fluentBitOptimized, configMap)
	return nil
}

func (cluster *EksCluster) InstallCloudMapController(refs core.BaseConstructSet, dag *core.ResourceGraph) (*kubernetes.KustomizeDirectory, error) {
	cloudMapController := &kubernetes.KustomizeDirectory{
		Name:          fmt.Sprintf("%s-cloudmap-controller", cluster.Name),
		ConstructRefs: refs,
		Directory:     "https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/config/controller_install_release",
		Cluster: &AwsResourceValue{
			ResourceVal: cluster,
			PropertyVal: CLUSTER_PROVIDER_IAC_VALUE,
		},
	}

	if controller := dag.GetResource(cloudMapController.Id()); controller != nil {
		if cm, ok := controller.(*kubernetes.KustomizeDirectory); ok {
			cloudMapController = cm
			cm.ConstructRefs.AddAll(refs)
		} else {
			return nil, errors.Errorf("Expected resource with id, %s, to be of type HelmChart, but was %s",
				controller.Id(), reflect.ValueOf(controller).Type().Name())
		}
	} else {
		clusterSetOutputPath := path.Join(MANIFEST_PATH_PREFIX, CM_CLUSTER_SET)
		content, err := fs.ReadFile(eksManifests, clusterSetOutputPath)
		if err != nil {
			return nil, err
		}
		clusterSet := &kubernetes.Manifest{
			Name:          fmt.Sprintf("%s-%s", cluster.Name, "cluster-set"),
			ConstructRefs: refs,
			FilePath:      clusterSetOutputPath,
			Content:       content,
			Transformations: map[string]core.IaCValue{
				`spec["value"]`: &AwsResourceValue{ResourceVal: cluster, PropertyVal: NAME_IAC_VALUE},
			},
			Cluster: &AwsResourceValue{
				ResourceVal: cluster,
				PropertyVal: CLUSTER_PROVIDER_IAC_VALUE,
			},
		}
		dag.AddResource(clusterSet)
		dag.AddDependenciesReflect(cloudMapController)
		dag.AddDependency(clusterSet, cloudMapController)
	}

	for _, nodeGroup := range cluster.GetClustersNodeGroups(dag) {
		dag.AddDependency(cloudMapController, nodeGroup)
	}

	return cloudMapController, nil
}

func (cluster *EksCluster) InstallAlbController(references core.BaseConstructSet, dag *core.ResourceGraph, appName string) (*kubernetes.HelmChart, error) {
	serviceAccountName := "aws-load-balancer-controller"
	saPath := "aws-load-balancer-controller-service-account.yaml"
	outputPath := path.Join(MANIFEST_PATH_PREFIX, saPath)
	serviceAccount := &kubernetes.ServiceAccount{Name: serviceAccountName}

	role, err := core.CreateResource[*IamRole](dag, RoleCreateParams{
		AppName: appName,
		Name:    "alb-controller",
		Refs:    references,
	})
	if err != nil {
		return nil, err
	}
	oidc, err := core.CreateResource[*OpenIdConnectProvider](dag, OidcCreateParams{
		AppName:     appName,
		ClusterName: strings.TrimLeft(cluster.Name, fmt.Sprintf("%s-", appName)),
		Refs:        role.ConstructsRef.Clone(),
	})
	var aRef core.BaseConstruct
	for _, ref := range references {
		aRef = ref
		break
	}
	dag.AddDependency(role, oidc)
	policy := createAlbControllerPolicy(cluster.Name, aRef)
	dag.AddDependency(role, policy)
	if err != nil {
		return nil, err
	}
	serviceAccount.Transformations[`metadata["annotations"]["eks.amazonaws.com/role-arn"]`] = &AwsResourceValue{ResourceVal: role, PropertyVal: ARN_IAC_VALUE}
	serviceAccount.FilePath = outputPath
	serviceAccount.Cluster = &AwsResourceValue{
		ResourceVal: cluster,
		PropertyVal: CLUSTER_PROVIDER_IAC_VALUE,
	}

	dag.AddDependenciesReflect(serviceAccount)

	albChart := &kubernetes.HelmChart{
		Name:          fmt.Sprintf("%s-alb-controller", cluster.Name),
		Chart:         "aws-load-balancer-controller",
		Repo:          "https://aws.github.io/eks-charts",
		ConstructRefs: references,
		Version:       "1.4.7",
		Namespace:     "default",
		Cluster: &AwsResourceValue{
			ResourceVal: cluster,
			PropertyVal: CLUSTER_PROVIDER_IAC_VALUE,
		},
		Values: map[string]any{
			"clusterName": AwsResourceValue{ResourceVal: cluster, PropertyVal: NAME_IAC_VALUE},
			"serviceAccount": map[string]any{
				"create": false,
				"name":   serviceAccount.Name,
			},
			"region": AwsResourceValue{ResourceVal: NewRegion(), PropertyVal: NAME_IAC_VALUE},
			"vpcId":  AwsResourceValue{ResourceVal: cluster.Vpc, PropertyVal: ID_IAC_VALUE},
			"podLabels": map[string]string{
				"app": "aws-lb-controller",
			},
			// objectSelector is used to select pods to inject the pod readiness gate into
			// (see https://kubernetes-sigs.github.io/aws-load-balancer-controller/v2.4/deploy/pod_readiness_gate/)
			"objectSelector": map[string]any{"matchLabels": map[string]any{"elbv2.k8s.aws/pod-readiness-gate-inject": "enabled"}},
			// webhookNamespaceSelector is set to an empty matchExpressions to allow the pod readiness gate to be installed in any namespace
			"webhookNamespaceSelectors": map[string]any{"matchExpressions": []any{}},
		},
	}
	dag.AddDependenciesReflect(albChart)
	dag.AddDependenciesReflect(role)
	for _, nodeGroup := range cluster.GetClustersNodeGroups(dag) {
		dag.AddDependency(albChart, nodeGroup)
	}
	return albChart, nil
}

func (cluster *EksCluster) installVpcCniAddon(references core.BaseConstructSet, dag *core.ResourceGraph) {
	addonName := "vpc-cni"
	addon := &EksAddon{
		Name:          fmt.Sprintf("%s-addon-%s", cluster.Name, addonName),
		ConstructsRef: references,
		AddonName:     addonName,
		ClusterName: &AwsResourceValue{
			ResourceVal: cluster,
			PropertyVal: NAME_IAC_VALUE,
		},
	}
	dag.AddDependenciesReflect(addon)
}

func (cluster *EksCluster) GetClustersNodeGroups(dag *core.ResourceGraph) []*EksNodeGroup {
	nodeGroups := []*EksNodeGroup{}
	for _, res := range dag.GetAllUpstreamResources(cluster) {
		if nodeGroup, ok := res.(*EksNodeGroup); ok {
			nodeGroups = append(nodeGroups, nodeGroup)
		}
	}
	return nodeGroups
}

func GetServiceAccountRole(sa *kubernetes.ServiceAccount, dag *core.ResourceGraph) (*IamRole, error) {
	roles := core.GetDownstreamResourcesOfType[*IamRole](dag, sa)
	if len(roles) > 1 {
		return nil, fmt.Errorf("service account %s has multiple roles", sa.Name)
	} else if len(roles) == 0 {
		return nil, fmt.Errorf("service account %s has no roles", sa.Name)
	}
	return roles[0], nil
}

func createEKSKubeconfig(cluster *EksCluster, region *Region) *kubernetes.Kubeconfig {
	clusterNameIaCValue := &AwsResourceValue{
		ResourceVal: cluster,
		PropertyVal: NAME_IAC_VALUE,
	}
	return &kubernetes.Kubeconfig{
		ConstructsRef:  cluster.ConstructsRef,
		Name:           fmt.Sprintf("%s-eks-kubeconfig", cluster.Name),
		ApiVersion:     "v1",
		CurrentContext: clusterNameIaCValue,
		Kind:           "Config",
		Clusters: []kubernetes.KubeconfigCluster{
			{
				Name: clusterNameIaCValue,
				Cluster: map[string]core.IaCValue{
					"certificate-authority-data": &AwsResourceValue{
						ResourceVal: cluster,
						PropertyVal: CLUSTER_CA_DATA_IAC_VALUE,
					},
					"server": &AwsResourceValue{
						ResourceVal: cluster,
						PropertyVal: CLUSTER_ENDPOINT_IAC_VALUE,
					},
				},
			},
		},
		Contexts: []kubernetes.KubeconfigContexts{
			{
				Name: clusterNameIaCValue,
				Context: kubernetes.KubeconfigContext{
					Cluster: clusterNameIaCValue,
					User:    clusterNameIaCValue,
				},
			},
		},
		Users: []kubernetes.KubeconfigUsers{
			{
				Name: clusterNameIaCValue,
				User: kubernetes.KubeconfigUser{
					Exec: kubernetes.KubeconfigExec{
						ApiVersion: "client.authentication.k8s.io/v1beta1",
						Command:    "aws",
						Args: []any{
							"eks",
							"get-token",
							"--cluster-name",
							clusterNameIaCValue,
							"--region",
							AwsResourceValue{
								ResourceVal: region,
								PropertyVal: NAME_IAC_VALUE,
							},
						},
					},
				},
			},
		},
	}
}

func GetServiceAccountAssumeRolePolicy(serviceAccountName string, oidc *OpenIdConnectProvider) *PolicyDocument {
	return &PolicyDocument{
		Version: VERSION,
		Statement: []StatementEntry{
			{
				Effect: "Allow",
				Principal: &Principal{
					Federated: &AwsResourceValue{
						ResourceVal: oidc,
						PropertyVal: ARN_IAC_VALUE,
					},
				},
				Action: []string{"sts:AssumeRoleWithWebIdentity"},
				Condition: &Condition{
					StringEquals: map[*AwsResourceValue]string{
						{
							ResourceVal: oidc,
							PropertyVal: OIDC_SUB_IAC_VALUE,
						}: fmt.Sprintf("system:serviceaccount:default:%s", k8sSanitizer.MetadataNameSanitizer.Apply(serviceAccountName)), // TODO: Replace default with the namespace when we expose via configuration
						{
							ResourceVal: oidc,
							PropertyVal: OIDC_AUD_IAC_VALUE,
						}: "sts.amazonaws.com",
					},
				},
			},
		},
	}
}

// BaseConstructsRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (cluster *EksCluster) BaseConstructsRef() core.BaseConstructSet {
	return cluster.ConstructsRef
}

// Id returns the id of the cloud resource
func (cluster *EksCluster) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     EKS_CLUSTER_TYPE,
		Name:     cluster.Name,
	}
}

func (c *EksCluster) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}

// BaseConstructsRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (addon *EksAddon) BaseConstructsRef() core.BaseConstructSet {
	return addon.ConstructsRef
}

// Id returns the id of the cloud resource
func (addon *EksAddon) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     EKS_ADDON_TYPE,
		Name:     addon.Name,
	}
}

func (addon *EksAddon) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}

// BaseConstructsRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (profile *EksFargateProfile) BaseConstructsRef() core.BaseConstructSet {
	return profile.ConstructsRef
}

// Id returns the id of the cloud resource
func (profile *EksFargateProfile) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     EKS_FARGATE_PROFILE_TYPE,
		Name:     profile.Name,
	}
}

func (profile *EksFargateProfile) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}

// BaseConstructsRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (group *EksNodeGroup) BaseConstructsRef() core.BaseConstructSet {
	return group.ConstructsRef
}

// Id returns the id of the cloud resource
func (group *EksNodeGroup) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     EKS_NODE_GROUP_TYPE,
		Name:     group.Name,
	}
}

func (group *EksNodeGroup) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}

func amiFromInstanceType(instanceType string) string {
	prefix := strings.Split(instanceType, ".")[0]
	for key, value := range EKS_AMI_INSTANCE_PREFIX_MAP {
		for _, toMatch := range value {
			if toMatch == prefix {
				return key
			}
		}
	}
	return ""
}

func createAlbControllerPolicy(clusterName string, ref core.BaseConstruct) *IamPolicy {
	policy := NewIamPolicy(clusterName, "alb-controller", ref, CreateAllowPolicyDocument([]string{
		"ec2:DescribeAccountAttributes",
		"ec2:DescribeAddresses",
		"ec2:DescribeAvailabilityZones",
		"ec2:DescribeInternetGateways",
		"ec2:DescribeVpcs",
		"ec2:DescribeVpcPeeringConnections",
		"ec2:DescribeSubnets",
		"ec2:DescribeSecurityGroups",
		"ec2:DescribeInstances",
		"ec2:DescribeNetworkInterfaces",
		"ec2:DescribeTags",
		"ec2:GetCoipPoolUsage",
		"ec2:DescribeCoipPools",
		"elasticloadbalancing:DescribeLoadBalancers",
		"elasticloadbalancing:DescribeLoadBalancerAttributes",
		"elasticloadbalancing:DescribeListeners",
		"elasticloadbalancing:DescribeListenerCertificates",
		"elasticloadbalancing:DescribeSSLPolicies",
		"elasticloadbalancing:DescribeRules",
		"elasticloadbalancing:DescribeTargetGroups",
		"elasticloadbalancing:DescribeTargetGroupAttributes",
		"elasticloadbalancing:DescribeTargetHealth",
		"elasticloadbalancing:DescribeTags",
		"elasticloadbalancing:CreateListener",
		"elasticloadbalancing:DeleteListener",
		"elasticloadbalancing:CreateRule",
		"elasticloadbalancing:DeleteRule",
	},
		[]*AwsResourceValue{{PropertyVal: core.ALL_RESOURCES_IAC_VALUE}},
	))
	policy.Policy.Statement = append(policy.Policy.Statement, StatementEntry{
		Effect: "Allow",
		Action: []string{
			"cognito-idp:DescribeUserPoolClient",
			"acm:ListCertificates",
			"acm:DescribeCertificate",
			"iam:ListServerCertificates",
			"iam:GetServerCertificate",
			"waf-regional:GetWebACL",
			"waf-regional:GetWebACLForResource",
			"waf-regional:AssociateWebACL",
			"waf-regional:DisassociateWebACL",
			"wafv2:GetWebACL",
			"wafv2:GetWebACLForResource",
			"wafv2:AssociateWebACL",
			"wafv2:DisassociateWebACL",
			"shield:GetSubscriptionState",
			"shield:DescribeProtection",
			"shield:CreateProtection",
			"shield:DeleteProtection",
		},
		Resource: []*AwsResourceValue{{PropertyVal: core.ALL_RESOURCES_IAC_VALUE}},
	})
	policy.Policy.Statement = append(policy.Policy.Statement, StatementEntry{
		Effect: "Allow",
		Action: []string{
			"iam:CreateServiceLinkedRole",
		},
		Resource: []*AwsResourceValue{{PropertyVal: core.ALL_RESOURCES_IAC_VALUE}},
		Condition: &Condition{StringEquals: map[*AwsResourceValue]string{
			{PropertyVal: "iam:AWSServiceName"}: "elasticloadbalancing.amazonaws.com",
		}},
	})
	policy.Policy.Statement = append(policy.Policy.Statement, StatementEntry{
		Effect: "Allow",
		Action: []string{
			"ec2:AuthorizeSecurityGroupIngress",
			"ec2:RevokeSecurityGroupIngress",
		},
		Resource: []*AwsResourceValue{{PropertyVal: core.ALL_RESOURCES_IAC_VALUE}},
	})
	policy.Policy.Statement = append(policy.Policy.Statement, StatementEntry{
		Effect: "Allow",
		Action: []string{
			"ec2:CreateSecurityGroup",
		},
		Resource: []*AwsResourceValue{{PropertyVal: core.ALL_RESOURCES_IAC_VALUE}},
	})
	policy.Policy.Statement = append(policy.Policy.Statement, StatementEntry{
		Effect: "Allow",
		Action: []string{
			"ec2:CreateTags",
		},
		Resource: []*AwsResourceValue{{PropertyVal: "arn:aws:ec2:*:*:security-group/*"}},
		Condition: &Condition{
			StringEquals: map[*AwsResourceValue]string{
				{PropertyVal: "ec2:CreateAction"}: "CreateSecurityGroup",
			},
			Null: map[*AwsResourceValue]string{
				{PropertyVal: "aws:RequestTag/elbv2.k8s.aws/cluster"}: "false",
			},
		},
	})
	policy.Policy.Statement = append(policy.Policy.Statement, StatementEntry{
		Effect: "Allow",
		Action: []string{
			"ec2:CreateTags",
			"ec2:DeleteTags",
		},
		Resource: []*AwsResourceValue{{PropertyVal: "arn:aws:ec2:*:*:security-group/*"}},
		Condition: &Condition{
			StringEquals: map[*AwsResourceValue]string{
				{PropertyVal: "ec2:CreateAction"}: "CreateSecurityGroup",
			},
			Null: map[*AwsResourceValue]string{
				{PropertyVal: "aws:RequestTag/elbv2.k8s.aws/cluster"}:  "true",
				{PropertyVal: "aws:ResourceTag/elbv2.k8s.aws/cluster"}: "false",
			},
		},
	})
	policy.Policy.Statement = append(policy.Policy.Statement, StatementEntry{
		Effect: "Allow",
		Action: []string{
			"ec2:AuthorizeSecurityGroupIngress",
			"ec2:RevokeSecurityGroupIngress",
			"ec2:DeleteSecurityGroup",
		},
		Resource: []*AwsResourceValue{{PropertyVal: "arn:aws:ec2:*:*:security-group/*"}},
		Condition: &Condition{
			Null: map[*AwsResourceValue]string{
				{PropertyVal: "aws:ResourceTag/elbv2.k8s.aws/cluster"}: "false",
			},
		},
	})
	policy.Policy.Statement = append(policy.Policy.Statement, StatementEntry{
		Effect: "Allow",
		Action: []string{
			"elasticloadbalancing:CreateLoadBalancer",
			"elasticloadbalancing:CreateTargetGroup",
		},
		Resource: []*AwsResourceValue{{PropertyVal: "arn:aws:ec2:*:*:security-group/*"}},
		Condition: &Condition{
			Null: map[*AwsResourceValue]string{
				{PropertyVal: "aws:RequestTag/elbv2.k8s.aws/cluster"}: "false",
			},
		},
	})
	policy.Policy.Statement = append(policy.Policy.Statement, StatementEntry{
		Effect: "Allow",
		Action: []string{
			"elasticloadbalancing:AddTags",
			"elasticloadbalancing:RemoveTags",
		},
		Resource: []*AwsResourceValue{
			{PropertyVal: "arn:aws:elasticloadbalancing:*:*:targetgroup/*/*"},
			{PropertyVal: "arn:aws:elasticloadbalancing:*:*:loadbalancer/net/*/*"},
			{PropertyVal: "arn:aws:elasticloadbalancing:*:*:loadbalancer/app/*/*"},
		},
		Condition: &Condition{
			Null: map[*AwsResourceValue]string{
				{PropertyVal: "aws:RequestTag/elbv2.k8s.aws/cluster"}:  "true",
				{PropertyVal: "aws:ResourceTag/elbv2.k8s.aws/cluster"}: "false",
			},
		},
	})
	policy.Policy.Statement = append(policy.Policy.Statement, StatementEntry{
		Effect: "Allow",
		Action: []string{
			"elasticloadbalancing:AddTags",
			"elasticloadbalancing:RemoveTags",
		},
		Resource: []*AwsResourceValue{
			{PropertyVal: "arn:aws:elasticloadbalancing:*:*:listener/net/*/*/*"},
			{PropertyVal: "arn:aws:elasticloadbalancing:*:*:listener/app/*/*/*"},
			{PropertyVal: "arn:aws:elasticloadbalancing:*:*:listener-rule/net/*/*/*"},
			{PropertyVal: "arn:aws:elasticloadbalancing:*:*:listener-rule/app/*/*/*"},
		},
	})
	policy.Policy.Statement = append(policy.Policy.Statement, StatementEntry{
		Effect: "Allow",
		Action: []string{
			"elasticloadbalancing:ModifyLoadBalancerAttributes",
			"elasticloadbalancing:SetIpAddressType",
			"elasticloadbalancing:SetSecurityGroups",
			"elasticloadbalancing:SetSubnets",
			"elasticloadbalancing:DeleteLoadBalancer",
			"elasticloadbalancing:ModifyTargetGroup",
			"elasticloadbalancing:ModifyTargetGroupAttributes",
			"elasticloadbalancing:DeleteTargetGroup",
		},
		Resource: []*AwsResourceValue{
			{PropertyVal: core.ALL_RESOURCES_IAC_VALUE},
		},
		Condition: &Condition{
			Null: map[*AwsResourceValue]string{
				{PropertyVal: "aws:RequestTag/elbv2.k8s.aws/cluster"}: "false",
			},
		},
	})
	policy.Policy.Statement = append(policy.Policy.Statement, StatementEntry{
		Effect: "Allow",
		Action: []string{
			"elasticloadbalancing:RegisterTargets",
			"elasticloadbalancing:DeregisterTargets",
		},
		Resource: []*AwsResourceValue{
			{PropertyVal: "arn:aws:elasticloadbalancing:*:*:targetgroup/*/*"},
		},
	})
	policy.Policy.Statement = append(policy.Policy.Statement, StatementEntry{
		Effect: "Allow",
		Action: []string{
			"elasticloadbalancing:SetWebAcl",
			"elasticloadbalancing:ModifyListener",
			"elasticloadbalancing:AddListenerCertificates",
			"elasticloadbalancing:RemoveListenerCertificates",
			"elasticloadbalancing:ModifyRule",
		},
		Resource: []*AwsResourceValue{
			{PropertyVal: core.ALL_RESOURCES_IAC_VALUE},
		},
	})
	return policy
}
