package constructs

import (
	"embed"
	"fmt"
	"gopkg.in/yaml.v3"
	"path/filepath"
	"strings"
)

//go:embed templates/*
var templates embed.FS

var cachedConstructs = make(map[ConstructTemplateId]ConstructTemplate)

func loadConstructTemplate(id ConstructTemplateId) (ConstructTemplate, error) {
	// Parse cachedConstructs from a template file
	if template, ok := cachedConstructs[id]; ok {
		return template, nil
	}

	if !strings.HasPrefix(id.Package, "klotho.") {
		return ConstructTemplate{}, fmt.Errorf("invalid package: %s", id.Package)
	}

	parentDir := strings.ToLower(strings.ReplaceAll(strings.SplitN(id.Package, ".", 2)[1], ".", "/"))
	constructKey := strings.ToLower(id.Name)
	// Read the YAML fileContent
	fileContent, err := templates.ReadFile(filepath.Join("templates", parentDir, constructKey, constructKey+".yaml"))
	if err != nil {
		return ConstructTemplate{}, fmt.Errorf("failed to read file: %v", err)
	}

	// Unmarshal the YAML fileContent into a map
	var template ConstructTemplate
	if err := yaml.Unmarshal(fileContent, &template); err != nil {
		return ConstructTemplate{}, fmt.Errorf("failed to unmarshal yaml: %v", err)
	}

	cachedConstructs[template.Id] = template

	return template, nil
}
