package aws

import (
	"fmt"
	"testing"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/infra/kubernetes"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/stretchr/testify/assert"
)

func Test_shouldCreateNetwork(t *testing.T) {
	cases := []struct {
		name       string
		constructs []core.Construct
		config     *config.Application
		want       bool
	}{
		{
			name:       "lambda",
			constructs: []core.Construct{&core.ExecutionUnit{}},
			config:     &config.Application{Defaults: config.Defaults{ExecutionUnit: config.KindDefaults{Type: Lambda}}},
			want:       false,
		},
		{
			name:       "kubernetes",
			constructs: []core.Construct{&core.ExecutionUnit{}},
			config:     &config.Application{Defaults: config.Defaults{ExecutionUnit: config.KindDefaults{Type: kubernetes.KubernetesType}}},
			want:       true,
		},
		{
			name:       "orm",
			constructs: []core.Construct{&core.Orm{}},
			want:       true,
		},
		{
			name:       "redis Node",
			constructs: []core.Construct{&core.RedisNode{}},
			want:       true,
		},
		{
			name:       "redis Cluster",
			constructs: []core.Construct{&core.RedisCluster{}},
			want:       true,
		},
		{
			name: "remaining resources",
			constructs: []core.Construct{
				&core.StaticUnit{},
				&core.Secrets{},
				&core.Fs{},
				&core.Kv{},
				&core.Config{},
				&core.InternalResource{},
				&core.Gateway{},
				&core.PubSub{},
			},
			want: false,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			aws := AWS{
				Config: tt.config,
			}
			result := core.NewConstructGraph()
			for _, construct := range tt.constructs {
				result.AddConstruct(construct)
			}
			should, err := aws.shouldCreateNetwork(result)
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.want, should)
		})

	}
}

func Test_createEksClusters(t *testing.T) {
	cases := []struct {
		name   string
		units  []*core.ExecutionUnit
		config *config.Application
		want   []*resources.EksCluster
	}{
		{
			name: `no clusters created`,
			units: []*core.ExecutionUnit{
				{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}},
			},
			config: &config.Application{
				AppName: "test",
				ExecutionUnits: map[string]*config.ExecutionUnit{
					"test": {Type: Lambda},
				},
			},
		},
		{
			name: `one exec unit, no cluster id`,
			units: []*core.ExecutionUnit{
				{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}},
			},
			config: &config.Application{
				AppName: "test",
				ExecutionUnits: map[string]*config.ExecutionUnit{
					"test": {Type: kubernetes.KubernetesType},
				},
			},
			want: []*resources.EksCluster{
				{Name: "test-eks-cluster", ConstructsRef: []core.AnnotationKey{
					{ID: "test", Capability: annotation.ExecutionUnitCapability},
				}},
			},
		},
		{
			name: `one exec unit, none eks`,
			units: []*core.ExecutionUnit{
				{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}},
			},
			config: &config.Application{
				AppName: "test",
			},
		},
		{
			name: `two eks units, unassigned`,
			units: []*core.ExecutionUnit{
				{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}},
				{AnnotationKey: core.AnnotationKey{ID: "test2", Capability: annotation.ExecutionUnitCapability}},
			},
			config: &config.Application{
				AppName: "test",
				ExecutionUnits: map[string]*config.ExecutionUnit{
					"test":  {Type: kubernetes.KubernetesType},
					"test2": {Type: kubernetes.KubernetesType},
				},
			},
			want: []*resources.EksCluster{
				{Name: "test-eks-cluster", ConstructsRef: []core.AnnotationKey{
					{ID: "test", Capability: annotation.ExecutionUnitCapability},
					{ID: "test2", Capability: annotation.ExecutionUnitCapability},
				}},
			},
		},
		{
			name: `two eks units, one unassigned`,
			units: []*core.ExecutionUnit{
				{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}},
				{AnnotationKey: core.AnnotationKey{ID: "test2", Capability: annotation.ExecutionUnitCapability}},
			},
			config: &config.Application{
				AppName: "test",
				ExecutionUnits: map[string]*config.ExecutionUnit{
					"test":  {Type: kubernetes.KubernetesType},
					"test2": {Type: kubernetes.KubernetesType, InfraParams: config.ConvertToInfraParams(config.KubernetesTypeParams{ClusterId: "cluster2"})},
				},
			},
			want: []*resources.EksCluster{
				{Name: "test-cluster2", ConstructsRef: []core.AnnotationKey{
					{ID: "test", Capability: annotation.ExecutionUnitCapability},
					{ID: "test2", Capability: annotation.ExecutionUnitCapability},
				}},
			},
		},
		{
			name: `two eks units, separate assignment`,
			units: []*core.ExecutionUnit{
				{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}},
				{AnnotationKey: core.AnnotationKey{ID: "test2", Capability: annotation.ExecutionUnitCapability}},
			},
			config: &config.Application{
				AppName: "test",
				ExecutionUnits: map[string]*config.ExecutionUnit{
					"test":  {Type: kubernetes.KubernetesType, InfraParams: config.ConvertToInfraParams(config.KubernetesTypeParams{ClusterId: "cluster1"})},
					"test2": {Type: kubernetes.KubernetesType, InfraParams: config.ConvertToInfraParams(config.KubernetesTypeParams{ClusterId: "cluster2"})},
				},
			},
			want: []*resources.EksCluster{
				{Name: "test-cluster1", ConstructsRef: []core.AnnotationKey{
					{ID: "test", Capability: annotation.ExecutionUnitCapability},
				}},
				{Name: "test-cluster2", ConstructsRef: []core.AnnotationKey{
					{ID: "test2", Capability: annotation.ExecutionUnitCapability},
				}},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			aws := AWS{
				Config: tt.config,
			}

			result := core.NewConstructGraph()
			for _, unit := range tt.units {
				result.AddConstruct(unit)
			}
			dag := core.NewResourceGraph()

			err := aws.createEksClusters(result, dag)
			if !assert.NoError(err) {
				return
			}
			numEksClusters := 0
			for _, res := range dag.ListResources() {
				if _, ok := res.(*resources.EksCluster); ok {
					numEksClusters++
				}
			}
			assert.Equal(numEksClusters, len(tt.want))

			for _, cluster := range tt.want {
				resource := dag.GetResource(cluster.Id())
				if !assert.NotNil(resource, fmt.Sprintf("Did not find cluster with id, %s", cluster.Id())) {
					return
				}
				assert.ElementsMatch(resource.KlothoConstructRef(), cluster.ConstructsRef)
			}

			if len(tt.want) > 0 {
				sg := resources.GetSecurityGroup(aws.Config, dag)
				assert.Contains(sg.IngressRules, resources.SecurityGroupRule{
					Description: "Allows ingress traffic from the EKS control plane",
					FromPort:    9443,
					Protocol:    "TCP",
					ToPort:      9443,
					CidrBlocks: []core.IaCValue{
						{Property: "0.0.0.0/0"},
					},
				})
			}
		})

	}
}
