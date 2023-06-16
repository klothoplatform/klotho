package resources

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider"
	corev1 "k8s.io/api/core/v1"
)

type (
	ServiceAccount struct {
		Name            string
		ConstructRefs   core.BaseConstructSet
		Object          corev1.Service
		Transformations map[string]core.IaCValue
		FilePath        string
	}
)

const (
	SERVICE_ACCOUNT_TYPE = "service_account"
)

// BaseConstructsRef returns a slice containing the ids of any Klotho constructs is correlated to
func (sa *ServiceAccount) BaseConstructsRef() core.BaseConstructSet { return sa.ConstructRefs }

func (sa *ServiceAccount) Id() core.ResourceId {
	return core.ResourceId{
		Provider: provider.KUBERNETES,
		Type:     SERVICE_ACCOUNT_TYPE,
		Name:     sa.Name,
	}
}
func (sa *ServiceAccount) OutputYAML() core.File {
	var outputFile core.File
	return outputFile
}
