package resources

import (
	"fmt"
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
)

var nodeGroupSanitizer = aws.EksNodeGroupSanitizer
var profileSanitizer = aws.EksFargateProfileSanitizer
var clusterSanitizer = aws.EksClusterSanitizer

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
// The method will create all of the corresponding IAM Roles necessary and attach all the execution units references to the following objects
func CreateEksCluster(appName string, clusterName string, subnets []*Subnet, securityGroups []*any, units []*core.ExecutionUnit, dag *core.ResourceGraph) {
	references := []core.AnnotationKey{}
	for _, u := range units {
		references = append(references, u.Provenance())
	}

	clusterRole := createClusterAdminRole(appName, clusterName+"-k8sAdmin", references)
	dag.AddResource(clusterRole)

	cluster := NewEksCluster(appName, clusterName, subnets, securityGroups, clusterRole)
	cluster.ConstructsRef = references
	dag.AddResource(cluster)
	dag.AddDependency(cluster, clusterRole)

	fargateRole := createPodExecutionRole(appName, clusterName+"-FargateExecutionRole", references)
	dag.AddResource(fargateRole)

	profile := NewEksFargateProfile(cluster, subnets, fargateRole, references)
	profile.Selectors = append(profile.Selectors, &FargateProfileSelector{Namespace: "default", Labels: map[string]string{"klotho-fargate-enabled": "true"}})

	dag.AddResource(profile)
	dag.AddDependency(profile, fargateRole)
	dag.AddDependency(profile, cluster)

	nodeRole := createNodeRole(appName, clusterName+"-NodeGroupRole", references)
	dag.AddResource(nodeRole)

	nodeGroup := NewEksNodeGroup(cluster, subnets, nodeRole, references)
	dag.AddResource(nodeGroup)
	dag.AddDependency(nodeGroup, nodeRole)
	dag.AddDependency(nodeGroup, cluster)

	for _, s := range subnets {
		dag.AddDependency(cluster, s)
		dag.AddDependency(nodeGroup, s)
		dag.AddDependency(profile, s)
	}

	for _, addOn := range createAddOns(clusterName, references) {
		dag.AddResource(addOn)
		dag.AddDependency(addOn, nodeGroup)
	}
}

func createAddOns(clusterName string, provenance []core.AnnotationKey) []*kubernetes.HelmChart {
	return []*kubernetes.HelmChart{
		&kubernetes.HelmChart{
			Name:          clusterName + `-metrics-server`,
			Chart:         "metrics-server",
			ConstructRefs: provenance,
			Repo:          `https://kubernetes-sigs.github.io/metrics-server/`,
		},
		&kubernetes.HelmChart{
			Name:             clusterName + `-cert-manager`,
			Chart:            `cert-manager`,
			ConstructRefs:    provenance,
			ClustersProvider: nil,
			Repo:             `https://charts.jetstack.io`,
			Version:          `v1.10.0`,
			Values: map[string]any{
				`installCRDs`: true,
				`webhook`: map[string]any{
					`timeoutSeconds`: 30,
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
			Resource: []core.IaCValue{core.IaCValue{Property: "*"}},
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

func NewEksNodeGroup(cluster *EksCluster, subnets []*Subnet, nodeRole *IamRole, ref []core.AnnotationKey) *EksNodeGroup {
	return &EksNodeGroup{
		Name:           nodeGroupSanitizer.Apply(cluster.Name),
		ConstructsRef:  ref,
		Cluster:        cluster,
		Subnets:        subnets,
		NodeRole:       nodeRole,
		DesiredSize:    2,
		MinSize:        1,
		MaxSize:        3,
		MaxUnavailable: 1,
		InstanceTypes:  []string{"t3.medium"},
	}
}

// Provider returns name of the provider the resource is correlated to
func (cluster *EksNodeGroup) Provider() string {
	return AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (cluster *EksNodeGroup) KlothoConstructRef() []core.AnnotationKey {
	return cluster.ConstructsRef
}

// ID returns the id of the cloud resource
func (cluster *EksNodeGroup) Id() string {
	return fmt.Sprintf("%s:%s:%s", cluster.Provider(), EKS_NODE_GROUP_TYPE, cluster.Name)
}
