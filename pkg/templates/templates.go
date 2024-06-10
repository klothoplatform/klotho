package templates

import "embed"

//go:embed */resources/*.yaml
var ResourceTemplates embed.FS

//go:embed */edges/*.yaml
var EdgeTemplates embed.FS

//go:embed */models/*.yaml  models/*.yaml
var Models embed.FS
