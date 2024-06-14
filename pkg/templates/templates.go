package templates

import (
	"embed"

	"github.com/klothoplatform/klotho/pkg/knowledgebase"
	"github.com/klothoplatform/klotho/pkg/knowledgebase/reader"
)

//go:embed */resources/*.yaml
var ResourceTemplates embed.FS

//go:embed */edges/*.yaml
var EdgeTemplates embed.FS

//go:embed */models/*.yaml  models/*.yaml
var Models embed.FS

func NewKBFromTemplates() (knowledgebase.TemplateKB, error) {
	return reader.NewKBFromFs(ResourceTemplates, EdgeTemplates, Models)
}
