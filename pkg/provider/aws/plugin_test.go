package aws

import (
	"fmt"
	"testing"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/stretchr/testify/assert"
)

func Test_createEksClusters(t *testing.T) {
	cases := []struct {
		name   string
		units  []*core.ExecutionUnit
		config *config.Application
		want   []*resources.EksCluster
	}{
		{
			name: `one exec unit, no cluster id`,
			units: []*core.ExecutionUnit{
				{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}},
			},
			config: &config.Application{
				AppName: "test",
				ExecutionUnits: map[string]*config.ExecutionUnit{
					"test": {Type: Kubernetes},
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
					"test":  {Type: Kubernetes},
					"test2": {Type: Kubernetes},
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
					"test":  {Type: Kubernetes},
					"test2": {Type: Kubernetes, InfraParams: config.ConvertToInfraParams(config.KubernetesTypeParams{ClusterId: "cluster2"})},
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
					"test":  {Type: Kubernetes, InfraParams: config.ConvertToInfraParams(config.KubernetesTypeParams{ClusterId: "cluster1"})},
					"test2": {Type: Kubernetes, InfraParams: config.ConvertToInfraParams(config.KubernetesTypeParams{ClusterId: "cluster2"})},
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
		})

	}
}
