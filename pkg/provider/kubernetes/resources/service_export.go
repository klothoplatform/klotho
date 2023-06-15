package resources

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider"
)

type (
	ServiceExport struct {
		Name          string
		ConstructRefs core.BaseConstructSet
		Service       *Service
		Namespace     *Namespace
		FilePath      string
	}
)

const (
	SERVICE_EXPORT_TYPE = "service_export"
)

func (se *ServiceExport) BaseConstructsRef() core.BaseConstructSet {
	return se.ConstructRefs
}

func (se *ServiceExport) Id() core.ResourceId {
	return core.ResourceId{
		Provider: provider.KUBERNETES,
		Type:     SERVICE_EXPORT_TYPE,
		Name:     se.Name,
	}
}

func (se *ServiceExport) OutputYAML() core.File {
	var outputFile core.File
	return outputFile
}
