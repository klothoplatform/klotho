package resources

import (
	"fmt"
	"math"
	"strings"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

const (
	EKS_CLUSTER_TYPE         = "eks_cluster"
	EKS_FARGATE_PROFILE_TYPE = "eks_fargate_profile"
	EKS_NODE_GROUP_TYPE      = "eks_node_group"
	DEFAULT_CLUSTER_NAME     = "eks-cluster"

	CLUSTER_OIDC_URL_IAC_VALUE = "cluster_oidc_url"
	CLUSTER_OIDC_ARN_IAC_VALUE = "cluster_oidc_arn"
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

type (
	//Todo: Add SecurityGroups when they are available
	EksCluster struct {
		Name          string
		ConstructsRef []core.AnnotationKey
		ClusterRole   *IamRole
		Subnets       []*Subnet
	}

	EksFargateProfile struct {
		Name             string
		ConstructsRef    []core.AnnotationKey
		Cluster          *EksCluster
		PodExecutionRole *IamRole
		Selectors        []*FargateProfileSelector `render:"template"`
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
// The method will create all of the corresponding IAM Roles necessary and attach all the execution units references to the following objects
func CreateEksCluster(cfg *config.Application, clusterName string, subnets []*Subnet, securityGroups []*any, units []*core.ExecutionUnit, dag *core.ResourceGraph) {
	references := []core.AnnotationKey{}
	for _, u := range units {
		references = append(references, u.Provenance())
	}

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
			spec = &groupSpec{}
			groupSpecs[key] = spec
		}
		spec.refs = append(spec.refs, unit.AnnotationKey)
		spec.DiskSizeGiB = int(math.Max(float64(spec.DiskSizeGiB), float64(spec.DiskSizeGiB)))
	}

	appName := cfg.AppName

	clusterRole := createClusterAdminRole(appName, clusterName+"-k8sAdmin", references)
	dag.AddResource(clusterRole)

	cluster := NewEksCluster(appName, clusterName, subnets, securityGroups, clusterRole)
	cluster.ConstructsRef = references
	dag.AddDependenciesReflect(cluster)

	for groupKey, spec := range groupSpecs {
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
		nodeGroup.NodeRole = createNodeRole(appName, fmt.Sprintf("%s.%s", clusterName, nodeGroup.Name), references)
		for _, sn := range subnets {
			if sn.Type == groupKey.NetworkType {
				nodeGroup.Subnets = append(nodeGroup.Subnets, sn)
			}
		}

		dag.AddDependenciesReflect(nodeGroup)
	}

	fargateRole := createPodExecutionRole(appName, clusterName+"-FargateExecutionRole", references)
	dag.AddResource(fargateRole)

	profile := NewEksFargateProfile(cluster, subnets, fargateRole, references)
	profile.Selectors = append(profile.Selectors, &FargateProfileSelector{Namespace: "default", Labels: map[string]string{"klotho-fargate-enabled": "true"}})
	dag.AddDependenciesReflect(profile)
}

func NodeGroupNameFromConfig(cfg config.ExecutionUnit) string {
	params := cfg.GetExecutionUnitParamsAsKubernetes()
	return NodeGroupName(cfg.NetworkPlacement, params.InstanceType)
}

func NodeGroupName(networkPlacement string, instanceType string) string {
	// ?? Does this need to handle instanceType == "" ?
	return nodeGroupSanitizer.Apply(fmt.Sprintf("%s_%s", networkPlacement, instanceType))
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
