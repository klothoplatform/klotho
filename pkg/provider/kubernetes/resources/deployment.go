package resources

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider"
	apps "k8s.io/api/apps/v1"
)

type (
	Deployment struct {
		ConstructRefs   core.BaseConstructSet
		Object          apps.Deployment
		Transformations map[string]core.IaCValue
		FilePath        string
	}
)

const (
	DEPLOYMENT_TYPE = "deployment"
)

func (deployment *Deployment) BaseConstructsRef() core.BaseConstructSet {
	return deployment.ConstructRefs
}

func (deployment *Deployment) Id() core.ResourceId {
	return core.ResourceId{
		Provider: provider.KUBERNETES,
		Type:     DEPLOYMENT_TYPE,
		Name:     deployment.Object.Name,
	}
}

func (deployment *Deployment) OutputYAML() core.File {
	var outputFile core.File
	return outputFile
}

func (deployment *Deployment) GetFunctionality() core.Functionality {
	return core.Compute
}

func (deployment *Deployment) Kind() string {
	return deployment.Object.Kind
}

func (deployment *Deployment) Path() string {
	return deployment.FilePath
}
