package resources

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"go.uber.org/zap"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/infra/kubernetes"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

const (
	EKS_CLUSTER_TYPE         = "eks_cluster"
	EKS_FARGATE_PROFILE_TYPE = "eks_fargate_profile"
	EKS_NODE_GROUP_TYPE      = "eks_node_group"
	DEFAULT_CLUSTER_NAME     = "eks-cluster"

	CLUSTER_OIDC_URL_IAC_VALUE = "cluster_oidc_url"
	CLUSTER_OIDC_ARN_IAC_VALUE = "cluster_oidc_arn"
	CLUSTER_ENDPOINT_IAC_VALUE = "cluster_endpoint"
	CLUSTER_CA_DATA_IAC_VALUE  = "cluster_certificate_authority_data"
	CLUSTER_PROVIDER_IAC_VALUE = "cluster_provider"
	NAME_IAC_VALUE             = "name"

	AWS_OBSERVABILITY_NS_PATH         = "aws_observability_namespace.yaml"
	AWS_OBSERVABILITY_CONFIG_MAP_PATH = "aws_observability_configmap.yaml"
	AMAZON_CLOUDWATCH_NS_PATH         = "amazon_cloudwatch_namespace.yaml"
	FLUENT_BIT_CLUSTER_INFO           = "fluent_bit_cluster_info.yaml"
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
	//Todo: Add SecurityGroups when they are available
	EksCluster struct {
		Name          string
		ConstructsRef []core.AnnotationKey
		ClusterRole   *IamRole
		Subnets       []*Subnet
		Manifests     []core.File
		Kubeconfig    *kubernetes.Kubeconfig
	}

	EksFargateProfile struct {
		Name             string
		ConstructsRef    []core.AnnotationKey
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
		ConstructsRef  []core.AnnotationKey
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
)

// CreateEksCluster will create a cluster in the subnets provided, with the attached additional security groups
//
// The method will also create a fargate profile in the default namespace and a single NodeGroup.
// The method will create all the corresponding IAM Roles necessary and attach all the execution units references to the following objects
func CreateEksCluster(cfg *config.Application, clusterName string, vpc *Vpc, securityGroups []*any, units []*core.ExecutionUnit, dag *core.ResourceGraph) error {
	references := []core.AnnotationKey{}
	for _, u := range units {
		references = append(references, u.Provenance())
	}

	appName := cfg.AppName

	clusterRole := createClusterAdminRole(appName, clusterName+"-k8sAdmin", references)
	dag.AddResource(clusterRole)

	subnets := vpc.GetVpcSubnets(dag)
	cluster := NewEksCluster(appName, clusterName, subnets, securityGroups, clusterRole)
	cluster.ConstructsRef = references
	dag.AddDependenciesReflect(cluster)

	nodeGroups := createNodeGroups(cfg, dag, units, clusterName, cluster, subnets)

	err := cluster.createFargateLogging(references, dag)
	if err != nil {
		zap.S().Warnf("Unable to set up Fargate logging manifests for cluster %s: %s", clusterName, err.Error())
	}
	err = cluster.installFluentBit(references, dag)
	if err != nil {
		zap.S().Warnf("Unable to set up fluent bit manifests for cluster %s: %s", clusterName, err.Error())
	}

	fargateRole := createPodExecutionRole(appName, clusterName+"-FargateExecutionRole", references)
	dag.AddDependenciesReflect(fargateRole)

	profile := NewEksFargateProfile(cluster, subnets, fargateRole, references)
	profile.Selectors = append(profile.Selectors, &FargateProfileSelector{Namespace: "default", Labels: map[string]string{"klotho-fargate-enabled": "true"}})
	dag.AddDependenciesReflect(profile)

	// Find the region a
	var region *Region
	region, err = findClusterRegion(dag, cluster)
	if err != nil {
		return err
	}
	cluster.Kubeconfig = createEKSKubeconfig(cluster, region)

	for _, addOn := range createAddOns(cluster, vpc, references) {
		dag.AddDependenciesReflect(addOn)
		for _, nodeGroup := range nodeGroups {
			dag.AddDependency(addOn, nodeGroup)
		}
	}

	return nil
}

func findClusterRegion(dag *core.ResourceGraph, cluster *EksCluster) (*Region, error) {
	var region *Region
	for _, res := range dag.GetAllDownstreamResources(cluster) {
		if r, ok := res.(*Region); ok {
			region = r
		}
	}
	if region == nil {
		return nil, fmt.Errorf("downstream region not found for EksCluster with id, %s", cluster.Id())
	}
	return region, nil
}

func createNodeGroups(cfg *config.Application, dag *core.ResourceGraph, units []*core.ExecutionUnit, clusterName string, cluster *EksCluster, subnets []*Subnet) []*EksNodeGroup {
	type groupKey struct {
		InstanceType string
		NetworkType  string
	}

	type groupSpec struct {
		DiskSizeGiB int
		refs        []core.AnnotationKey
	}

	groupSpecs := make(map[groupKey]*groupSpec)

	for _, unit := range units {
		unitCfg := cfg.GetExecutionUnit(unit.ID)
		params := unitCfg.GetExecutionUnitParamsAsKubernetes()
		key := groupKey{InstanceType: params.InstanceType, NetworkType: unitCfg.NetworkPlacement}
		spec := groupSpecs[key]
		if spec == nil {
			spec = &groupSpec{
				DiskSizeGiB: 20,
			}
			groupSpecs[key] = spec
		}
		spec.refs = append(spec.refs, unit.AnnotationKey)
		if params.DiskSizeGiB > spec.DiskSizeGiB {
			spec.DiskSizeGiB = params.DiskSizeGiB
		}
	}

	var groups []*EksNodeGroup

	hasInstanceType := false
	for gk := range groupSpecs {
		if gk.InstanceType != "" {
			hasInstanceType = true
			break
		}
	}
	if !hasInstanceType {
		for gk, spec := range groupSpecs {
			if gk.InstanceType == "" {
				groupSpecs[groupKey{InstanceType: "t3.medium", NetworkType: gk.NetworkType}] = spec
			}
		}
	}

	for groupKey, spec := range groupSpecs {
		if groupKey.InstanceType == "" {
			continue
		}
		nodeGroup := &EksNodeGroup{
			Name:          NodeGroupName(groupKey.NetworkType, groupKey.InstanceType),
			ConstructsRef: spec.refs,
			Cluster:       cluster,
			DiskSize:      spec.DiskSizeGiB,
			AmiType:       amiFromInstanceType(groupKey.InstanceType),
			InstanceTypes: []string{groupKey.InstanceType},
			Labels: map[string]string{
				"network_placement": groupKey.NetworkType,
			},
			// TODO make these configurable
			DesiredSize:    2,
			MaxSize:        2,
			MinSize:        1,
			MaxUnavailable: 1,
		}
		nodeGroup.NodeRole = createNodeRole(cfg.AppName, fmt.Sprintf("%s.%s", clusterName, nodeGroup.Name), spec.refs)
		dag.AddDependenciesReflect(nodeGroup.NodeRole)

		for _, sn := range subnets {
			if sn.Type == groupKey.NetworkType {
				nodeGroup.Subnets = append(nodeGroup.Subnets, sn)
			}
		}

		dag.AddDependenciesReflect(nodeGroup)

		groups = append(groups, nodeGroup)
	}

	return groups
}

func NodeGroupNameFromConfig(cfg config.ExecutionUnit) string {
	params := cfg.GetExecutionUnitParamsAsKubernetes()
	return NodeGroupName(cfg.NetworkPlacement, params.InstanceType)
}

func NodeGroupName(networkPlacement string, instanceType string) string {
	return nodeGroupSanitizer.Apply(fmt.Sprintf("%s_%s", networkPlacement, instanceType))
}

func createAddOns(cluster *EksCluster, vpc *Vpc, provenance []core.AnnotationKey) []core.Resource {
	clustersProvider := core.IaCValue{
		Resource: cluster.Kubeconfig,
		Property: CLUSTER_PROVIDER_IAC_VALUE,
	}
	return []core.Resource{
		&kubernetes.HelmChart{
			Name:             cluster.Name + `-metrics-server`,
			Chart:            "metrics-server",
			ConstructRefs:    provenance,
			ClustersProvider: clustersProvider,
			Repo:             `https://kubernetes-sigs.github.io/metrics-server/`,
		},
		&kubernetes.HelmChart{
			Name:             cluster.Name + `-cert-manager`,
			Chart:            `cert-manager`,
			ConstructRefs:    provenance,
			ClustersProvider: clustersProvider,
			Repo:             `https://charts.jetstack.io`,
			Version:          `v1.10.0`,
			Values: map[string]any{
				`installCRDs`: true,
				`webhook`: map[string]any{
					`timeoutSeconds`: 30,
				},
			},
		},
		&HelmChartAlbController{
			HelmChart: kubernetes.HelmChart{

				Name:             cluster.Name + `-alb-c`,
				Chart:            `aws-load-balancer-controller`,
				ConstructRefs:    provenance,
				Repo:             `https://aws.github.io/eks-charts`,
				Version:          `1.4.7`,
				ClustersProvider: clustersProvider,
				Namespace:        "default", // kube-system if on fargate, default otherwise
			},
			ClusterName: cluster.Name,
			Region:      NewRegion(),
			Vpc:         vpc,
			Role:        NewIamRole(cluster.Name, `alb-controller-role`, provenance, ServiceRolePolicyDocument(cluster, `alb-controller`)),
		},
	}
}

func ServiceRolePolicyDocument(cluster *EksCluster, name string) *PolicyDocument {
	return &PolicyDocument{
		Version: VERSION,
		Statement: []StatementEntry{
			{
				Effect: "Allow",
				Principal: &Principal{
					Federated: core.IaCValue{
						Resource: cluster,
						Property: CLUSTER_OIDC_ARN_IAC_VALUE,
					},
				},
				Action: []string{"sts:AssumeRoleWithWebIdentity"},
				Condition: &Condition{
					StringEquals: map[core.IaCValue]string{
						{
							Resource: cluster,
							Property: CLUSTER_OIDC_URL_IAC_VALUE,
						}: fmt.Sprintf("system:serviceaccount:default:%s", name), // TODO: Replace default with the namespace when we expose via configuration
					},
				},
			},
		},
	}
}

func (cluster *EksCluster) GetOutputFiles() []core.File {
	return cluster.Manifests
}

func (cluster *EksCluster) createFargateLogging(references []core.AnnotationKey, dag *core.ResourceGraph) error {
	namespaceOutputPath := path.Join(MANIFEST_PATH_PREFIX, AWS_OBSERVABILITY_NS_PATH)
	content, err := fs.ReadFile(eksManifests, namespaceOutputPath)
	if err != nil {
		return err
	}
	namespace := &kubernetes.Manifest{
		Name:          fmt.Sprintf("%s-%s", cluster.Name, "aws-observability-ns"),
		ConstructRefs: references,
		FilePath:      namespaceOutputPath,
	}
	dag.AddResource(namespace)
	dag.AddDependency(namespace, cluster)
	cluster.Manifests = append(cluster.Manifests, &core.RawFile{FPath: namespaceOutputPath, Content: content})

	configMapOutputPath := path.Join(MANIFEST_PATH_PREFIX, AWS_OBSERVABILITY_CONFIG_MAP_PATH)
	content, err = fs.ReadFile(eksManifests, configMapOutputPath)
	if err != nil {
		return err
	}
	configMap := &kubernetes.Manifest{
		Name:          fmt.Sprintf("%s-%s", cluster.Name, "aws-observability-config-map"),
		ConstructRefs: references,
		FilePath:      configMapOutputPath,
	}
	dag.AddResource(configMap)
	dag.AddDependency(configMap, cluster)
	dag.AddDependency(configMap, namespace)
	cluster.Manifests = append(cluster.Manifests, &core.RawFile{FPath: configMapOutputPath, Content: content})
	return nil
}

func (cluster *EksCluster) installFluentBit(references []core.AnnotationKey, dag *core.ResourceGraph) error {
	namespaceOutputPath := path.Join(MANIFEST_PATH_PREFIX, AMAZON_CLOUDWATCH_NS_PATH)
	content, err := fs.ReadFile(eksManifests, namespaceOutputPath)
	if err != nil {
		return err
	}
	namespace := &kubernetes.Manifest{
		Name:          fmt.Sprintf("%s-%s", cluster.Name, "awmazon-cloudwatch-ns"),
		ConstructRefs: references,
		FilePath:      namespaceOutputPath,
	}
	dag.AddResource(namespace)
	dag.AddDependency(namespace, cluster)
	cluster.Manifests = append(cluster.Manifests, &core.RawFile{FPath: namespaceOutputPath, Content: content})

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
		Transformations: map[string]core.IaCValue{
			"data.cluster.name": {Resource: cluster, Property: NAME_IAC_VALUE},
			"data.logs.region":  {Resource: region, Property: NAME_IAC_VALUE},
		},
	}
	dag.AddResource(configMap)
	dag.AddDependency(configMap, cluster)
	dag.AddDependency(configMap, namespace)
	dag.AddDependency(configMap, region)
	cluster.Manifests = append(cluster.Manifests, &core.RawFile{FPath: configMapOutputPath, Content: content})

	fluentBitOptimized := &kubernetes.Manifest{
		Name:          fmt.Sprintf("%s-%s", cluster.Name, "fluent-bit"),
		ConstructRefs: references,
		FilePath:      "https://raw.githubusercontent.com/aws-samples/amazon-cloudwatch-container-insights/latest/k8s-deployment-manifest-templates/deployment-mode/daemonset/container-insights-monitoring/fluent-bit/fluent-bit.yaml",
	}
	dag.AddResource(configMap)
	dag.AddDependency(fluentBitOptimized, cluster)
	dag.AddDependency(fluentBitOptimized, configMap)
	return nil
}

func createEKSKubeconfig(cluster *EksCluster, region *Region) *kubernetes.Kubeconfig {
	username := "aws"
	clusterNameIaCValue := core.IaCValue{
		Resource: cluster,
		Property: NAME_IAC_VALUE,
	}
	return &kubernetes.Kubeconfig{
		ConstructsRef:  cluster.ConstructsRef,
		Name:           fmt.Sprintf("%s-eks-kubeconfig", cluster.Name),
		ApiVersion:     "v1",
		CurrentContext: "aws",
		Kind:           "Config",
		Clusters: []kubernetes.KubeconfigCluster{
			{
				Name: clusterNameIaCValue,
				CertificateAuthorityData: core.IaCValue{
					Resource: cluster,
					Property: CLUSTER_CA_DATA_IAC_VALUE,
				},
				Server: core.IaCValue{
					Resource: cluster,
					Property: CLUSTER_ENDPOINT_IAC_VALUE,
				},
			},
		},
		Contexts: []kubernetes.KubeconfigContext{
			{
				Cluster: clusterNameIaCValue,
				User:    username,
			},
		},
		Users: []kubernetes.KubeconfigUser{
			{
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
							Resource: region,
							Property: NAME_IAC_VALUE,
						},
					},
				},
			},
		},
	}
}

// GetEksCluster will return the resource with the name corresponding to the appName and ClusterId
//
// If the dag does not contain the resource or the resource is not an EksCluster, it will return nil
func GetEksCluster(appName string, clusterId string, dag *core.ResourceGraph) *EksCluster {
	if clusterId == "" {
		clusterId = DEFAULT_CLUSTER_NAME
	}
	cluster := NewEksCluster(appName, clusterId, nil, nil, nil)
	resource := dag.GetResource(cluster.Id())
	if existingCluster, ok := resource.(*EksCluster); ok {
		return existingCluster
	}
	return nil
}

func createClusterAdminRole(appName string, roleName string, refs []core.AnnotationKey) *IamRole {
	clusterRole := NewIamRole(appName, roleName, refs, EKS_ASSUME_ROLE_POLICY)
	clusterRole.AddAwsManagedPolicies([]string{"arn:aws:iam::aws:policy/AmazonEKSClusterPolicy"})
	return clusterRole
}

func createPodExecutionRole(appName string, roleName string, refs []core.AnnotationKey) *IamRole {
	fargateRole := NewIamRole(appName, roleName, refs, EKS_FARGATE_ASSUME_ROLE_POLICY)
	fargateRole.InlinePolicy = &PolicyDocument{Version: VERSION, Statement: []StatementEntry{
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
	}}
	return fargateRole
}

func createNodeRole(appName string, roleName string, refs []core.AnnotationKey) *IamRole {
	nodeRole := NewIamRole(appName, roleName, refs, EC2_ASSUMER_ROLE_POLICY)
	nodeRole.AddAwsManagedPolicies([]string{
		"arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy",
		"arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly",
		"arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy", "arn:aws:iam::aws:policy/AWSCloudMapFullAccess",
		"arn:aws:iam::aws:policy/CloudWatchAgentServerPolicy",
	})
	return nodeRole
}

func NewEksCluster(appName string, clusterName string, subnets []*Subnet, securityGroups []*any, role *IamRole) *EksCluster {
	return &EksCluster{
		Name:    clusterSanitizer.Apply(fmt.Sprintf("%s-%s", appName, clusterName)),
		Subnets: subnets,
		// SecurityGroups: securityGroups,
		ClusterRole: role,
	}
}

// Provider returns name of the provider the resource is correlated to
func (cluster *EksCluster) Provider() string {
	return AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (cluster *EksCluster) KlothoConstructRef() []core.AnnotationKey {
	return cluster.ConstructsRef
}

// ID returns the id of the cloud resource
func (cluster *EksCluster) Id() string {
	return fmt.Sprintf("%s:%s:%s", cluster.Provider(), EKS_CLUSTER_TYPE, cluster.Name)
}

func NewEksFargateProfile(cluster *EksCluster, subnets []*Subnet, nodeRole *IamRole, ref []core.AnnotationKey) *EksFargateProfile {
	return &EksFargateProfile{
		Name:             profileSanitizer.Apply(cluster.Name),
		ConstructsRef:    ref,
		Subnets:          subnets,
		Cluster:          cluster,
		PodExecutionRole: nodeRole,
	}
}

// Provider returns name of the provider the resource is correlated to
func (profile *EksFargateProfile) Provider() string {
	return AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (profile *EksFargateProfile) KlothoConstructRef() []core.AnnotationKey {
	return profile.ConstructsRef
}

// ID returns the id of the cloud resource
func (profile *EksFargateProfile) Id() string {
	return fmt.Sprintf("%s:%s:%s", profile.Provider(), EKS_FARGATE_PROFILE_TYPE, profile.Name)
}

// Provider returns name of the provider the resource is correlated to
func (group *EksNodeGroup) Provider() string {
	return AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (group *EksNodeGroup) KlothoConstructRef() []core.AnnotationKey {
	return group.ConstructsRef
}

// ID returns the id of the cloud resource
func (group *EksNodeGroup) Id() string {
	return fmt.Sprintf("%s:%s:%s", group.Provider(), EKS_NODE_GROUP_TYPE, group.Name)
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

type HelmChartAlbController struct {
	kubernetes.HelmChart

	// The following are specific to this struct
	ClusterName string
	Region      *Region
	Vpc         *Vpc
	Role        *IamRole
}
