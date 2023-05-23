package resources

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/infra/kubernetes"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func Test_InstallCloudMapController(t *testing.T) {
	assert := assert.New(t)
	dag := core.NewResourceGraph()
	cluster := NewEksCluster("test", "cluster", nil, nil, nil, nil)
	nodeGroup1 := &EksNodeGroup{
		Name:    "nodegroup1",
		Cluster: cluster,
	}
	dag.AddDependenciesReflect(nodeGroup1)

	unit1 := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "1"}}
	_, err := cluster.InstallCloudMapController(unit1.AnnotationKey, dag)
	if !assert.NoError(err) {
		return
	}

	cloudMapController := &kubernetes.KustomizeDirectory{
		Name: fmt.Sprintf("%s-cloudmap-controller", cluster.Name),
	}

	if controller := dag.GetResource(cloudMapController.Id()); controller != nil {
		if cm, ok := controller.(*kubernetes.KustomizeDirectory); ok {
			assert.Equal(cm.ClustersProvider, core.IaCValue{
				Resource: cluster,
				Property: CLUSTER_PROVIDER_IAC_VALUE,
			})
			assert.Equal(cm.Directory, "https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/config/controller_install_release")
		} else {
			assert.NoError(errors.Errorf("Expected resource with id, %s, to be of type HelmChart, but was %s",
				controller.Id(), reflect.ValueOf(controller).Type().Name()))
		}
	}

	unit2 := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "2"}}
	_, err = cluster.InstallCloudMapController(unit2.AnnotationKey, dag)
	if !assert.NoError(err) {
		return
	}

	if controller := dag.GetResource(cloudMapController.Id()); controller != nil {
		assert.ElementsMatch(controller.KlothoConstructRef(), []core.AnnotationKey{unit1.AnnotationKey, unit2.AnnotationKey})
	}

}

func Test_getClustersNodeGroups(t *testing.T) {
	assert := assert.New(t)
	dag := core.NewResourceGraph()
	cluster := NewEksCluster("test", "cluster", nil, nil, nil, nil)
	nodeGroup1 := &EksNodeGroup{
		Name:    "nodegroup1",
		Cluster: cluster,
	}
	nodeGroup2 := &EksNodeGroup{
		Name:    "nodegroup2",
		Cluster: cluster,
	}
	nodeGroup3 := &EksNodeGroup{
		Name: "nodegroup3",
	}
	dag.AddDependenciesReflect(nodeGroup1)
	dag.AddDependenciesReflect(nodeGroup2)
	dag.AddDependenciesReflect(nodeGroup3)
	assert.ElementsMatch(cluster.GetClustersNodeGroups(dag), []*EksNodeGroup{nodeGroup1, nodeGroup2})
}
