package templates

import "embed"

//go:embed aws/resources/*.yaml
var ResourceTemplates embed.FS

//go:embed aws/edges/*.yaml
var EdgeTemplates embed.FS
