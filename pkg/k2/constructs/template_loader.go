package constructs

import (
	"embed"
	"gopkg.in/yaml.v3"
	"log"
	"strings"
)

//go:embed templates/*
var templates embed.FS

var cachedConstructs = make(map[ConstructTemplateId]ConstructTemplate)

func loadConstructTemplate(id ConstructTemplateId) ConstructTemplate {
	// Parse cachedConstructs from a template file
	if template, ok := cachedConstructs[id]; ok {
		return template
	}

	if !strings.HasPrefix(id.Package, "klotho.") {
		panic("Invalid package")
	}

	// Read the YAML fileContent
	fileContent, err := templates.ReadFile("templates/aws/container/container.yaml")
	if err != nil {
		panic(err)
	}

	// Unmarshal the YAML fileContent into a map
	var template ConstructTemplate
	if err := yaml.Unmarshal(fileContent, &template); err != nil {
		log.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	cachedConstructs[template.Id] = template

	return template
}
