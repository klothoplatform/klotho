package templates

import "embed"

//go:embed aws/resources/*.yaml
var ResourceTemplates embed.FS
