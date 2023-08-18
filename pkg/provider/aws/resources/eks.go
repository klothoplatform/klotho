package resources

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"reflect"
	"regexp"
	"strings"

	"github.com/klothoplatform/klotho/pkg/engine/classification"
	corev1 "k8s.io/api/core/v1"
	k8sResource "k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/klothoplatform/klotho/pkg/core"
	kubernetes "github.com/klothoplatform/klotho/pkg/provider/kubernetes/resources"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
	k8sSanitizer "github.com/klothoplatform/klotho/pkg/sanitization/kubernetes"
	"github.com/pkg/errors"
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
	CLUSTER_EFS_RESOURCE_TAG_IAC_VALUE            = "efs_cluster_resource_tag"
	NAME_IAC_VALUE                                = "name"
	ID_IAC_VALUE                                  = "id"
	AWS_OBSERVABILITY_CONFIG_MAP_REGION_IAC_VALUE = "aws_observ_cm_region"
	NODE_GROUP_NAME_IAC_VALUE                     = "node_group_name"
	AWS_EFS_PERSISTENT_VOLUME_FILENAME            = "persistent_volume.yaml"
	AWS_EFS_STORAGECLASS_FILENAME                 = "storageclass.yaml"
	AWS_EFS_CLAIM_FILENAME                        = "claim.yaml"
	AWS_OBSERVABILITY_NS_PATH                     = "aws_observability_namespace.yaml"
	AWS_OBSERVABILITY_CONFIG_MAP_PATH             = "aws_observability_configmap.yaml"
	AMAZON_CLOUDWATCH_NS_PATH                     = "amazon_cloudwatch_namespace.yaml"
	FLUENT_BIT_CLUSTER_INFO                       = "fluent_bit_cluster_info.yaml"
	CM_CLUSTER_SET                                = "cloudmap_cluster_set.yaml"
	MANIFEST_PATH_PREFIX                          = "manifests"
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
		ConstructRefs  core.BaseConstructSet `yaml:"-"`
		ClusterRole    *IamRole
		Vpc            *Vpc
		Subnets        []*Subnet
		SecurityGroups []*SecurityGroup
		Kubeconfig     *kubernetes.Kubeconfig `yaml:"-"`
	}

	EksFargateProfile struct {
		Name             string
		ConstructRefs    core.BaseConstructSet `yaml:"-"`
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
		ConstructRefs  core.BaseConstructSet `yaml:"-"`
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
		ConstructRefs core.BaseConstructSet `yaml:"-"`
		AddonName     string
		ClusterName   core.IaCValue
		Role          *IamRole
	}
)

var (
	EKS_ANNOTATION_KEY = "eks.amazonaws.com/role-arn"
)
var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9 ]+`)

func sanitizeString(str string) string {
	return nonAlphanumericRegex.ReplaceAllString(str, "")
}
func GenerateRoleArnPlaceholder(unit string) string {
	return fmt.Sprintf("%sRoleArn", sanitizeString(unit))
}

func GenerateImagePlaceholder(unit string) string {
	return k8sSanitizer.RFC1123LabelSanitizer.Apply(fmt.Sprintf("%sImage", sanitizeString(unit)))
}

func GenerateTargetGroupBindingPlaceholder(unit string) string {
	return fmt.Sprintf("%sTargetGroupArn", sanitizeString(unit))
}

func GenerateInstanceTypeKeyPlaceholder(unit string) string {
	return fmt.Sprintf("%sInstanceTypeKey", sanitizeString(unit))
}

func GenerateInstanceTypeValuePlaceholder(unit string) string {
	return fmt.Sprintf("%sInstanceTypeValue", sanitizeString(unit))
}

func GeneratePersistentVolumeHandlePlaceholder(unit string) string {
	return fmt.Sprintf("%sVolumeHandle", sanitizeString(unit))
}

type EksClusterCreateParams struct {
	Refs    core.BaseConstructSet
	AppName string
	Name    string
}

func (cluster *EksCluster) Create(dag *core.ResourceGraph, params EksClusterCreateParams) error {

	cluster.Name = clusterSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	cluster.ConstructRefs = params.Refs.Clone()
	existingCluster := dag.GetResource(cluster.Id())
	if existingCluster != nil {
		graphCluster := existingCluster.(*EksCluster)
		graphCluster.ConstructRefs.AddAll(params.Refs)
	} else {
		dag.AddResource(cluster)

		// We create these add ons in cluster creation since there is edge which would create them
		// These are always installed in every cluster, no matter the configuration
		cluster.installVpcCniAddon(cluster.ConstructRefs, dag)
	}
	return nil
}

type EksClusterConfigureParams struct {
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
		existingProfile.ConstructRefs.AddAll(params.Refs)
	} else {
		profile.ConstructRefs = params.Refs.Clone()
		dag.AddResource(profile)
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
		existingNodeGroup.ConstructRefs.AddAll(params.Refs)
	} else {
		nodeGroup.ConstructRefs = params.Refs.Clone()
		nodeGroup.InstanceTypes = []string{params.InstanceType}
		nodeGroup.Labels = map[string]string{
			"network_placement": params.NetworkType,
		}
		dag.AddResource(nodeGroup)
	}

	return nil
}

type EksNodeGroupConfigureParams struct {
}

func (nodeGroup *EksNodeGroup) Configure(params EksNodeGroupConfigureParams) error {
	if len(nodeGroup.InstanceTypes) != 0 {
		nodeGroup.AmiType = amiFromInstanceType(nodeGroup.InstanceTypes[0])
	}
	return nil
}

func (cluster *EksCluster) SetUpDefaultNodeGroup(dag *core.ResourceGraph, appName string) error {
	ng, err := core.CreateResource[*EksNodeGroup](dag, EksNodeGroupCreateParams{
		InstanceType: "t3.medium",
		NetworkType:  PrivateSubnet,
		Refs:         cluster.ConstructRefs,
		AppName:      cluster.Name,
	})
	if err != nil {
		return err
	}
	dag.AddDependency(ng, cluster)
	cluster.CreatePrerequisiteCharts(dag)
	err = cluster.InstallFluentBit(cluster.ConstructRefs, dag)
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
			ConstructRefs: cluster.ConstructRefs,
			Cluster:       cluster.Id(),
			Repo:          `https://kubernetes-sigs.github.io/metrics-server/`,
			IsInternal:    true,
		},
		{
			Name:          cluster.Name + `-cert-manager`,
			Chart:         `cert-manager`,
			ConstructRefs: cluster.ConstructRefs,

			Cluster: cluster.Id(),
			Repo:    `https://charts.jetstack.io`,
			Version: `v1.12.0`,
			Values: map[string]any{
				`installCRDs`: true,
				`webhook`: map[string]any{
					`timeoutSeconds`: 30,
				},
			},
			IsInternal: true,
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
		Cluster:  cluster.Id(),
	}
	dag.AddDependenciesReflect(manifest)

	for _, ng := range cluster.GetClustersNodeGroups(dag) {
		dag.AddDependency(manifest, ng)
		if strings.HasSuffix(strings.ToLower(ng.AmiType), "_gpu") {
			manifest.ConstructRefs.AddAll(ng.ConstructRefs)
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
		Cluster:       cluster.Id(),
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
		Cluster:       cluster.Id(),
		Transformations: map[string]core.IaCValue{
			`data["output.conf"]`: core.IaCValue{ResourceId: cluster.Id(), Property: AWS_OBSERVABILITY_CONFIG_MAP_REGION_IAC_VALUE},
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
		Cluster:       cluster.Id(),
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
			`data["cluster.name"]`: core.IaCValue{ResourceId: cluster.Id(), Property: NAME_IAC_VALUE},
			`data["logs.region"]`:  core.IaCValue{ResourceId: region.Id(), Property: NAME_IAC_VALUE},
		},
		Cluster: cluster.Id(),
	}
	dag.AddResource(configMap)
	dag.AddDependency(configMap, cluster)
	dag.AddDependency(configMap, namespace)
	fluentBitOptimized := &kubernetes.Manifest{
		Name:          fmt.Sprintf("%s-%s", cluster.Name, "fluent-bit"),
		ConstructRefs: references,
		FilePath:      "https://raw.githubusercontent.com/aws-samples/amazon-cloudwatch-container-insights/latest/k8s-deployment-manifest-templates/deployment-mode/daemonset/container-insights-monitoring/fluent-bit/fluent-bit.yaml",
		Cluster:       cluster.Id(),
	}
	dag.AddResource(configMap)
	dag.AddDependency(fluentBitOptimized, cluster)
	dag.AddDependency(fluentBitOptimized, configMap)
	return nil
}

func CreatePersistentVolume(resource core.Resource, fileSystem *EfsFileSystem, dag *core.ResourceGraph, appName string) (*kubernetes.PersistentVolume, error) {
	cluster, err := core.GetSingleDownstreamResourceOfType[*EksCluster](dag, resource)
	if err != nil {
		return nil, err
	}

	// Create the PersistentVolume
	pv, err := core.CreateResource[*kubernetes.PersistentVolume](dag, kubernetes.PersistentVolumeCreateParams{
		Name:          fmt.Sprintf("%s-%s", resource.Id().Name, fileSystem.Id().Name),
		ConstructRefs: core.BaseConstructSetOf(resource, fileSystem),
	})
	if err != nil {
		return nil, err
	}
	pv.Cluster = cluster.Id()
	pv.Object.Spec.Capacity = corev1.ResourceList{"storage": k8sResource.MustParse("5Gi")}
	volumeMode := corev1.PersistentVolumeFilesystem
	pv.Object.Spec.VolumeMode = &volumeMode
	pv.Object.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany}
	pv.Object.Spec.PersistentVolumeReclaimPolicy = corev1.PersistentVolumeReclaimRetain
	if pv.Values == nil {
		pv.Values = make(map[string]core.IaCValue)
	}
	value := GeneratePersistentVolumeHandlePlaceholder(fileSystem.Name)
	volumeHandlePlaceholder := fmt.Sprintf("{{ .Values.%s }}", value)
	pv.Values[value] = core.IaCValue{ResourceId: fileSystem.Id(), Property: ID_IAC_VALUE}
	pv.Object.Spec.CSI = &corev1.CSIPersistentVolumeSource{
		Driver:       "efs.csi.aws.com",
		VolumeHandle: volumeHandlePlaceholder,
		VolumeAttributes: map[string]string{
			"encryptInTransit": "true",
		},
	}

	// Create the volume's PersistentVolumeClaim
	pvc, err := core.CreateResource[*kubernetes.PersistentVolumeClaim](dag, kubernetes.PersistentVolumeClaimCreateParams{
		Name:          fmt.Sprintf("%s-%s", resource.Id().Name, fileSystem.Id().Name),
		ConstructRefs: core.BaseConstructSetOf(resource, fileSystem),
	})
	if err != nil {
		return nil, err
	}
	pvc.Cluster = cluster.Id()
	pvc.Object.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany}
	pvc.Object.Spec.Resources.Requests = corev1.ResourceList{"storage": k8sResource.MustParse("5Gi")}

	// Create the volume's StorageClass
	sc, err := core.CreateResource[*kubernetes.StorageClass](dag, kubernetes.StorageClassCreateParams{
		Name:          fmt.Sprintf("%s-%s", resource.Id().Name, fileSystem.Id().Name),
		ConstructRefs: core.BaseConstructSetOf(resource, fileSystem),
	})
	if err != nil {
		return nil, err
	}
	sc.Cluster = cluster.Id()
	sc.Object.Provisioner = "efs.csi.aws.com"
	sc.Object.MountOptions = []string{"tls"}

	// Create the path from the upstream resource to the filesystem
	dag.AddDependency(resource, pv)
	dag.AddDependency(pv, pvc)
	dag.AddDependency(pv, sc)
	dag.AddDependency(pvc, sc)
	dag.AddDependency(pv, fileSystem)

	// Associate the volume and it's dependencies with the resource's downstream cluster
	dag.AddDependency(pv, cluster)
	dag.AddDependency(pvc, cluster)
	dag.AddDependency(sc, cluster)

	// Install the EFS CSI driver on the cluster if the pod is in an EKS node group
	oidc, err := core.GetSingleUpstreamResourceOfType[*OpenIdConnectProvider](dag, cluster)
	if err != nil {
		return nil, err
	}
	for _, downstream := range dag.GetDownstreamResources(resource) {
		// install the EFS CSI driver on the cluster if the pod is in an EKS node group
		if _, ok := downstream.(*EksNodeGroup); ok {
			if _, err := cluster.InstallEfsCsiDriverAddon(resource.BaseConstructRefs().CloneWith(fileSystem.BaseConstructRefs()), dag, appName, oidc); err != nil {
				return nil, err
			}
			break
		}
	}

	return pv, nil
}

func (cluster *EksCluster) InstallEfsCsiDriverAddon(references core.BaseConstructSet, dag *core.ResourceGraph, appName string, oidc *OpenIdConnectProvider) (*EksAddon, error) {
	addonName := "aws-efs-csi-driver"

	addonRole, err := core.CreateResource[*IamRole](dag, RoleCreateParams{
		AppName: appName,
		Name:    addonName,
		Refs:    references.Clone(),
	})
	if err != nil {
		return nil, err
	}
	addonRole.AddAwsManagedPolicies([]string{"arn:aws:iam::aws:policy/service-role/AmazonEFSCSIDriverPolicy"})

	// Allow the accounts created by the addon to assume the role
	addonRole.AssumeRolePolicyDoc = &PolicyDocument{
		Version: VERSION,
		Statement: []StatementEntry{
			{
				Effect: "Allow",
				Principal: &Principal{
					Federated: core.IaCValue{
						ResourceId: oidc.Id(),
						Property:   ARN_IAC_VALUE,
					},
				},
				Action: []string{"sts:AssumeRoleWithWebIdentity"},
				Condition: &Condition{
					StringLike: map[core.IaCValue]string{
						{
							ResourceId: oidc.Id(),
							Property:   OIDC_SUB_IAC_VALUE,
						}: "system:serviceaccount:kube-system:efs-csi-*",
						{
							ResourceId: oidc.Id(),
							Property:   OIDC_AUD_IAC_VALUE,
						}: "sts.amazonaws.com",
					},
				},
			},
		},
	}

	addon := &EksAddon{
		Name:          fmt.Sprintf("%s-addon-%s", cluster.Name, addonName),
		ConstructRefs: references,
		AddonName:     addonName,
		ClusterName: core.IaCValue{
			ResourceId: cluster.Id(),
			Property:   NAME_IAC_VALUE,
		},
		Role: addonRole,
	}
	dag.AddDependenciesReflect(addon)
	return addon, nil
}
func (cluster *EksCluster) InstallCloudMapController(refs core.BaseConstructSet, dag *core.ResourceGraph) (*kubernetes.KustomizeDirectory, error) {
	cloudMapController := &kubernetes.KustomizeDirectory{
		Name:          fmt.Sprintf("%s-cloudmap-controller", cluster.Name),
		ConstructRefs: refs,
		Directory:     "https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/config/controller_install_release",
		Cluster:       cluster.Id(),
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
				`spec["value"]`: core.IaCValue{ResourceId: cluster.Id(), Property: NAME_IAC_VALUE},
			},
			Cluster: cluster.Id(),
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

type ServiceAccountCreateParams struct {
	AppName    string
	Dag        *core.ResourceGraph
	Name       string
	Policy     *IamPolicy
	References core.BaseConstructSet
}

// CreateServiceAccount creates a service account for the cluster
func (cluster *EksCluster) CreateServiceAccount(params ServiceAccountCreateParams) (*kubernetes.ServiceAccount, error) {
	outputPath := path.Join(MANIFEST_PATH_PREFIX, fmt.Sprintf("%s-service-account.yaml", params.Name))
	serviceAccount := &kubernetes.ServiceAccount{
		Name: params.Name,
		Object: &corev1.ServiceAccount{
			TypeMeta: v1.TypeMeta{APIVersion: "v1", Kind: "ServiceAccount"},
		},
	}
	dag := params.Dag
	appName := params.AppName

	role, err := core.CreateResource[*IamRole](dag, RoleCreateParams{
		AppName: params.AppName,
		Name:    params.Name,
		Refs:    params.References.Clone(),
	})
	if err != nil {
		return nil, err
	}

	serviceAccount.FilePath = outputPath
	serviceAccount.Cluster = cluster.Id()

	oidc, err := core.CreateResource[*OpenIdConnectProvider](dag, OidcCreateParams{
		AppName:     appName,
		ClusterName: cluster.Name,
	})
	if err != nil {
		return nil, err
	}
	oidc.ConstructRefs.AddAll(params.References)

	dag.AddDependency(role, oidc)
	dag.AddDependency(role, params.Policy)
	dag.AddDependency(serviceAccount, role)
	dag.AddDependenciesReflect(serviceAccount)

	return serviceAccount, nil
}

func (cluster *EksCluster) InstallAlbController(references core.BaseConstructSet, dag *core.ResourceGraph, appName string) (*kubernetes.HelmChart, error) {
	if cluster.Vpc == nil {
		return nil, errors.Errorf("cluster.Vpc is required to install the alb controller")
	}
	clusterCharts := core.GetUpstreamResourcesOfType[*kubernetes.HelmChart](dag, cluster)
	var certManagerchart *kubernetes.HelmChart
	for _, chart := range clusterCharts {
		if chart.Chart == "cert-manager" && chart.Repo == "https://charts.jetstack.io" {
			certManagerchart = chart
			break
		}
	}
	if certManagerchart == nil {
		return nil, errors.Errorf("cert-manager chart is required to install the alb controller")
	}

	serviceAccountName := "aws-load-balancer-controller"
	var aRef core.BaseConstruct
	for _, r := range references {
		aRef = r
		break
	}
	serviceAccount, err := cluster.CreateServiceAccount(ServiceAccountCreateParams{
		AppName:    appName,
		Dag:        dag,
		Name:       serviceAccountName,
		Policy:     createAlbControllerPolicy(cluster.Name, aRef),
		References: references,
	})
	if err != nil {
		return nil, err
	}

	region := NewRegion()

	albChart := &kubernetes.HelmChart{
		Name:          fmt.Sprintf("%s-alb-controller", cluster.Name),
		Chart:         "aws-load-balancer-controller",
		Repo:          "https://aws.github.io/eks-charts",
		ConstructRefs: references,
		Version:       "1.5.5",
		Cluster:       cluster.Id(),
		IsInternal:    true,
		Values: map[string]any{
			"clusterName": core.IaCValue{ResourceId: cluster.Id(), Property: NAME_IAC_VALUE},
			"serviceAccount": map[string]any{
				"create": false,
				"name":   serviceAccount.Name,
			},
			"region": core.IaCValue{ResourceId: region.Id(), Property: NAME_IAC_VALUE},
			"vpcId":  core.IaCValue{ResourceId: cluster.Vpc.Id(), Property: ID_IAC_VALUE},
			// objectSelector is used to select pods to inject the pod readiness gate into
			// (see https://kubernetes-sigs.github.io/aws-load-balancer-controller/v2.4/deploy/pod_readiness_gate/)
			"objectSelector": map[string]any{"matchLabels": map[string]any{"elbv2.k8s.aws/pod-readiness-gate-inject": "enabled"}},
			// webhookNamespaceSelector is set to an empty matchExpressions to allow the pod readiness gate to be installed in any namespace
			"webhookNamespaceSelectors": map[string]any{"matchExpressions": []any{}},
			"enableCertManager":         true,
		},
	}
	albChart.Values["podLabels"] = map[string]string{
		"app":                      "aws-lb-controller",
		kubernetes.KLOTHO_ID_LABEL: k8sSanitizer.LabelValueSanitizer.Apply(albChart.Id().String()),
	}

	dag.AddResource(region)
	dag.AddDependenciesReflect(albChart)
	dag.AddDependency(albChart, serviceAccount)
	dag.AddDependency(albChart, certManagerchart)
	for _, nodeGroup := range cluster.GetClustersNodeGroups(dag) {
		dag.AddDependency(albChart, nodeGroup)
	}
	return albChart, nil
}

func (cluster *EksCluster) installVpcCniAddon(references core.BaseConstructSet, dag *core.ResourceGraph) {
	addonName := "vpc-cni"
	addon := &EksAddon{
		Name:          fmt.Sprintf("%s-addon-%s", cluster.Name, addonName),
		ConstructRefs: references,
		AddonName:     addonName,
		ClusterName: core.IaCValue{
			ResourceId: cluster.Id(),
			Property:   NAME_IAC_VALUE,
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
	if sa == nil {
		return nil, fmt.Errorf("service account is nil")
	}
	roles := core.GetDownstreamResourcesOfType[*IamRole](dag, sa)
	if len(roles) > 1 {
		return nil, fmt.Errorf("service account %s has multiple roles", sa.Name)
	} else if len(roles) == 0 {
		if sa.Cluster.IsZero() {
			return nil, fmt.Errorf("%s has no cluster", sa.Id())
		}

		role, err := core.CreateResource[*IamRole](dag, RoleCreateParams{
			Name: fmt.Sprintf("%s-Role", sa.Name),
			Refs: core.BaseConstructSetOf(sa),
		})
		if err != nil {
			return nil, err
		}
		dag.AddDependency(sa, role)
		if sa.Object == nil {
			return nil, fmt.Errorf("service account %s has no object", sa.Name)
		}

		value := GenerateRoleArnPlaceholder(role.Name)
		roleArnPlaceholder := fmt.Sprintf("{{ .Values.%s }}", value)

		if sa.Object.Annotations == nil {
			sa.Object.Annotations = make(map[string]string)
		}
		sa.Object.Annotations["eks.amazonaws.com/role-arn"] = roleArnPlaceholder

		if sa.Values == nil {
			sa.Values = make(map[string]core.IaCValue)
		}
		sa.Values[value] = core.IaCValue{ResourceId: role.Id(), Property: ID_IAC_VALUE}

		// Sets the role's AssumeRolePolicyDocument to allow the service account to assume the role
		oidc, err := core.CreateResource[*OpenIdConnectProvider](dag, OidcCreateParams{
			ClusterName: sa.Cluster.Name,
			Refs:        core.BaseConstructSetOf(sa),
		})
		if err != nil {
			return nil, err
		}
		assumeRolePolicy := GetServiceAccountAssumeRolePolicy(sa.Object.Name, sa.Object.Namespace, oidc)
		role.AssumeRolePolicyDoc = assumeRolePolicy
		dag.AddDependenciesReflect(role)
		return role, nil
	}
	return roles[0], nil
}

func configureKubeconfig(cluster *EksCluster, region *Region) error {
	kubeconfig := cluster.Kubeconfig
	if kubeconfig == nil {
		return fmt.Errorf("kubeconfig for cluster %s is nil", cluster.Id())
	}

	clusterNameIaCValue := core.IaCValue{
		ResourceId: cluster.Id(),
		Property:   NAME_IAC_VALUE,
	}
	kubeconfig.ApiVersion = "v1"
	kubeconfig.CurrentContext = clusterNameIaCValue
	kubeconfig.Kind = "Config"
	kubeconfig.Clusters = []kubernetes.KubeconfigCluster{
		{
			Name: clusterNameIaCValue,
			Cluster: map[string]core.IaCValue{
				"certificate-authority-data": core.IaCValue{
					ResourceId: cluster.Id(),
					Property:   CLUSTER_CA_DATA_IAC_VALUE,
				},
				"server": core.IaCValue{
					ResourceId: cluster.Id(),
					Property:   CLUSTER_ENDPOINT_IAC_VALUE,
				},
			},
		},
	}
	kubeconfig.Contexts = []kubernetes.KubeconfigContexts{
		{
			Name: clusterNameIaCValue,
			Context: kubernetes.KubeconfigContext{
				Cluster: clusterNameIaCValue,
				User:    clusterNameIaCValue,
			},
		},
	}
	kubeconfig.Users = []kubernetes.KubeconfigUsers{
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
						core.IaCValue{
							ResourceId: region.Id(),
							Property:   NAME_IAC_VALUE,
						},
					},
				},
			},
		},
	}
	return nil
}

func GetServiceAccountAssumeRolePolicy(serviceAccountName string, namespace string, oidc *OpenIdConnectProvider) *PolicyDocument {
	if namespace == "" {
		namespace = "default"
	}
	return &PolicyDocument{
		Version: VERSION,
		Statement: []StatementEntry{
			{
				Effect: "Allow",
				Principal: &Principal{
					Federated: core.IaCValue{
						ResourceId: oidc.Id(),
						Property:   ARN_IAC_VALUE,
					},
				},
				Action: []string{"sts:AssumeRoleWithWebIdentity"},
				Condition: &Condition{
					StringEquals: map[core.IaCValue]string{
						{
							ResourceId: oidc.Id(),
							Property:   OIDC_SUB_IAC_VALUE,
						}: fmt.Sprintf("system:serviceaccount:%s:%s", namespace, serviceAccountName),
						{
							ResourceId: oidc.Id(),
							Property:   OIDC_AUD_IAC_VALUE,
						}: "sts.amazonaws.com",
					},
				},
			},
		},
	}
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (cluster *EksCluster) BaseConstructRefs() core.BaseConstructSet {
	return cluster.ConstructRefs
}

// Id returns the id of the cloud resource
func (cluster *EksCluster) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     EKS_CLUSTER_TYPE,
		Name:     cluster.Name,
	}
}

func (cluster *EksCluster) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (addon *EksAddon) BaseConstructRefs() core.BaseConstructSet {
	return addon.ConstructRefs
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

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (profile *EksFargateProfile) BaseConstructRefs() core.BaseConstructSet {
	return profile.ConstructRefs
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

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (group *EksNodeGroup) BaseConstructRefs() core.BaseConstructSet {
	return group.ConstructRefs
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
	/*

	 */

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
		[]core.IaCValue{{Property: core.ALL_RESOURCES_IAC_VALUE}},
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
		Resource: []core.IaCValue{{Property: core.ALL_RESOURCES_IAC_VALUE}},
	})
	policy.Policy.Statement = append(policy.Policy.Statement, StatementEntry{
		Effect: "Allow",
		Action: []string{
			"iam:CreateServiceLinkedRole",
		},
		Resource: []core.IaCValue{{Property: core.ALL_RESOURCES_IAC_VALUE}},
		Condition: &Condition{StringEquals: map[core.IaCValue]string{
			{Property: "iam:AWSServiceName"}: "elasticloadbalancing.amazonaws.com",
		}},
	})
	policy.Policy.Statement = append(policy.Policy.Statement, StatementEntry{
		Effect: "Allow",
		Action: []string{
			"ec2:AuthorizeSecurityGroupIngress",
			"ec2:RevokeSecurityGroupIngress",
		},
		Resource: []core.IaCValue{{Property: core.ALL_RESOURCES_IAC_VALUE}},
	})
	policy.Policy.Statement = append(policy.Policy.Statement, StatementEntry{
		Effect: "Allow",
		Action: []string{
			"ec2:CreateSecurityGroup",
		},
		Resource: []core.IaCValue{{Property: core.ALL_RESOURCES_IAC_VALUE}},
	})
	policy.Policy.Statement = append(policy.Policy.Statement, StatementEntry{
		Effect: "Allow",
		Action: []string{
			"ec2:CreateTags",
		},
		Resource: []core.IaCValue{{Property: "arn:aws:ec2:*:*:security-group/*"}},
		Condition: &Condition{
			StringEquals: map[core.IaCValue]string{
				{Property: "ec2:CreateAction"}: "CreateSecurityGroup",
			},
			Null: map[core.IaCValue]string{
				{Property: "aws:RequestTag/elbv2.k8s.aws/cluster"}: "false",
			},
		},
	})
	policy.Policy.Statement = append(policy.Policy.Statement, StatementEntry{
		Effect: "Allow",
		Action: []string{
			"ec2:CreateTags",
			"ec2:DeleteTags",
		},
		Resource: []core.IaCValue{{Property: "arn:aws:ec2:*:*:security-group/*"}},
		Condition: &Condition{
			StringEquals: map[core.IaCValue]string{
				{Property: "ec2:CreateAction"}: "CreateSecurityGroup",
			},
			Null: map[core.IaCValue]string{
				{Property: "aws:RequestTag/elbv2.k8s.aws/cluster"}:  "true",
				{Property: "aws:ResourceTag/elbv2.k8s.aws/cluster"}: "false",
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
		Resource: []core.IaCValue{{Property: "arn:aws:ec2:*:*:security-group/*"}},
		Condition: &Condition{
			Null: map[core.IaCValue]string{
				{Property: "aws:ResourceTag/elbv2.k8s.aws/cluster"}: "false",
			},
		},
	})
	policy.Policy.Statement = append(policy.Policy.Statement, StatementEntry{
		Effect: "Allow",
		Action: []string{
			"elasticloadbalancing:CreateLoadBalancer",
			"elasticloadbalancing:CreateTargetGroup",
		},
		Resource: []core.IaCValue{{Property: "arn:aws:ec2:*:*:security-group/*"}},
		Condition: &Condition{
			Null: map[core.IaCValue]string{
				{Property: "aws:RequestTag/elbv2.k8s.aws/cluster"}: "false",
			},
		},
	})
	policy.Policy.Statement = append(policy.Policy.Statement, StatementEntry{
		Effect: "Allow",
		Action: []string{
			"elasticloadbalancing:AddTags",
			"elasticloadbalancing:RemoveTags",
		},
		Resource: []core.IaCValue{
			{Property: "arn:aws:elasticloadbalancing:*:*:targetgroup/*/*"},
			{Property: "arn:aws:elasticloadbalancing:*:*:loadbalancer/net/*/*"},
			{Property: "arn:aws:elasticloadbalancing:*:*:loadbalancer/app/*/*"},
		},
		Condition: &Condition{
			Null: map[core.IaCValue]string{
				{Property: "aws:RequestTag/elbv2.k8s.aws/cluster"}:  "true",
				{Property: "aws:ResourceTag/elbv2.k8s.aws/cluster"}: "false",
			},
		},
	})
	policy.Policy.Statement = append(policy.Policy.Statement, StatementEntry{
		Effect: "Allow",
		Action: []string{
			"elasticloadbalancing:AddTags",
			"elasticloadbalancing:RemoveTags",
		},
		Resource: []core.IaCValue{
			{Property: "arn:aws:elasticloadbalancing:*:*:listener/net/*/*/*"},
			{Property: "arn:aws:elasticloadbalancing:*:*:listener/app/*/*/*"},
			{Property: "arn:aws:elasticloadbalancing:*:*:listener-rule/net/*/*/*"},
			{Property: "arn:aws:elasticloadbalancing:*:*:listener-rule/app/*/*/*"},
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
		Resource: []core.IaCValue{
			{Property: core.ALL_RESOURCES_IAC_VALUE},
		},
		Condition: &Condition{
			Null: map[core.IaCValue]string{
				{Property: "aws:RequestTag/elbv2.k8s.aws/cluster"}: "false",
			},
		},
	})
	policy.Policy.Statement = append(policy.Policy.Statement, StatementEntry{
		Effect: "Allow",
		Action: []string{
			"elasticloadbalancing:RegisterTargets",
			"elasticloadbalancing:DeregisterTargets",
		},
		Resource: []core.IaCValue{
			{Property: "arn:aws:elasticloadbalancing:*:*:targetgroup/*/*"},
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
		Resource: []core.IaCValue{
			{Property: core.ALL_RESOURCES_IAC_VALUE},
		},
	})
	return policy
}

func (cluster *EksCluster) MakeOperational(dag *core.ResourceGraph, appName string, classifier classification.Classifier) error {
	var podsAndDeployments []core.Resource
	var nodeGroups []*EksNodeGroup
	var fargateProfiles []*EksFargateProfile

	// We create these add-ons in cluster creation since there is edge which would create them
	// These are always installed in every cluster, no matter the configuration
	cluster.installVpcCniAddon(cluster.ConstructRefs, dag)

	for _, downstream := range dag.GetAllUpstreamResources(cluster) {
		switch downstream := downstream.(type) {
		case *EksNodeGroup:
			nodeGroups = append(nodeGroups, downstream)
		case *EksFargateProfile:
			fargateProfiles = append(fargateProfiles, downstream)
		case *kubernetes.Pod:
			podsAndDeployments = append(podsAndDeployments, downstream)
		case *kubernetes.Deployment:
			podsAndDeployments = append(podsAndDeployments, downstream)
		}
	}

	cluster.associateDeployablesToDeploymentTargets(dag, podsAndDeployments, fargateProfiles, nodeGroups)

	// Add the kubeconfig after the dependencies are added otherwise we will have a circular dependency
	if err := configureKubeconfig(cluster, NewRegion()); err != nil {
		return err
	}
	return nil
}

func (cluster *EksCluster) associateDeployablesToDeploymentTargets(dag *core.ResourceGraph, podsAndDeployments []core.Resource, fargateProfiles []*EksFargateProfile, nodeGroups []*EksNodeGroup) {
	var deployedTo core.Resource
	for _, deployable := range podsAndDeployments {
		if deployedTo != nil {
			break
		}
		for _, resource := range dag.GetDownstreamResources(deployable) {
			switch resource.(type) {
			case *EksFargateProfile:
				deployedTo = resource
			case *EksNodeGroup:
				deployedTo = resource
			}
		}

		if deployedTo == nil {
			if len(fargateProfiles) > 0 {
				deployedTo = fargateProfiles[0]
			} else if len(nodeGroups) > 0 {
				deployedTo = nodeGroups[0]
			}
		}
		if deployedTo != nil {
			for _, pod := range podsAndDeployments {
				dag.AddDependency(pod, deployedTo)
			}
		}
		dag.AddDependency(deployable, deployedTo)
	}
}
