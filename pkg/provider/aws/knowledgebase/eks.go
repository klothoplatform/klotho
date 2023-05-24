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
				CidrBlocks: []core.IaCValue{
					{Property: "0.0.0.0/0"},
				},
			})
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.EksFargateProfile, *resources.Subnet]{},
	knowledgebase.EdgeBuilder[*resources.EksFargateProfile, *resources.EksCluster]{
		Expand: func(profile *resources.EksFargateProfile, cluster *resources.EksCluster, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if len(cluster.GetClustersNodeGroups(dag)) == 0 {
				_, err := core.CreateResource[*resources.EksNodeGroup](dag, resources.EksNodeGroupCreateParams{
					InstanceType: "t3.medium",
					NetworkType:  resources.PrivateSubnet,
					Refs:         cluster.ConstructsRef,
					AppName:      data.AppName,
					ClusterName:  strings.TrimLeft(cluster.Name, fmt.Sprintf("%s-", data.AppName)),
				})
				if err != nil {
					return err
				}
				cluster.CreatePrerequisiteCharts(dag)
				err = cluster.InstallFluentBit(cluster.ConstructsRef, dag)
				if err != nil {
					return err
				}
			}
			err := cluster.CreateFargateLogging(profile.ConstructsRef, dag)
			return err
		},
	},
	knowledgebase.EdgeBuilder[*resources.EksNodeGroup, *resources.EksCluster]{
		Expand: func(nodeGroup *resources.EksNodeGroup, cluster *resources.EksCluster, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
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
	knowledgebase.EdgeBuilder[*kubernetes.Manifest, *resources.EksCluster]{},
	knowledgebase.EdgeBuilder[*kubernetes.Manifest, *resources.EksFargateProfile]{},
	knowledgebase.EdgeBuilder[*kubernetes.Manifest, *resources.EksNodeGroup]{},
	knowledgebase.EdgeBuilder[*kubernetes.Manifest, *kubernetes.Manifest]{},
	knowledgebase.EdgeBuilder[*kubernetes.Manifest, *resources.Region]{},
)
