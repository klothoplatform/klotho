package constructs

import (
	"embed"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

//go:embed templates/*
var templates embed.FS

var (
	cachedConstructs = make(map[ConstructTemplateId]ConstructTemplate)
	cachedBindings   = make(map[string]BindingTemplate)
	mu               sync.Mutex
)

func loadConstructTemplate(id ConstructTemplateId) (ConstructTemplate, error) {
	mu.Lock()
	defer mu.Unlock()
	if template, ok := cachedConstructs[id]; ok {

		return template, nil
	}

	if !strings.HasPrefix(id.Package, "klotho.") {
		return ConstructTemplate{}, fmt.Errorf("invalid package: %s", id.Package)
	}

	constructDir, err := getConstructTemplateDir(id)
	if err != nil {
		return ConstructTemplate{}, err
	}
	constructKey := strings.ToLower(id.Name)

	fileContent, err := templates.ReadFile(filepath.Join(constructDir, constructKey+".yaml"))
	if err != nil {
		return ConstructTemplate{}, fmt.Errorf("failed to read file: %w", err)
	}

	var template ConstructTemplate
	if err := yaml.Unmarshal(fileContent, &template); err != nil {
		return ConstructTemplate{}, fmt.Errorf("failed to unmarshal yaml: %w", err)
	}

	cachedConstructs[template.Id] = template

	return template, nil
}

func loadBindingTemplate(owner ConstructTemplateId, from ConstructTemplateId, to ConstructTemplateId) (BindingTemplate, error) {
	mu.Lock()
	defer mu.Unlock()
	if owner != from && owner != to {
		return BindingTemplate{}, fmt.Errorf("owner must be either from or to")
	}
	// binding key name depends on whether the owner is from or to
	// if the owner is from, the key is to_<to_name>
	// if the owner is to, the key is from_<from_name>
	// this is because the binding template is stored in the directory of the owner
	// and each binding may have a separate template file for both the from and to constructs
	var bindingKey string
	if owner == from {
		bindingKey = "to_" + to.String()
	} else {
		bindingKey = "from_" + from.String()
	}

	cacheKey := fmt.Sprintf("%s/%s", owner.String(), bindingKey)

	if template, ok := cachedBindings[cacheKey]; ok {
		return template, nil
	}

	constructDir, err := getConstructTemplateDir(owner)
	if err != nil {
		return BindingTemplate{}, err
	}
	bindingsDir := filepath.Join(constructDir, "bindings")

	// Read the YAML fileContent
	fileContent, err := templates.ReadFile(filepath.Join(bindingsDir, bindingKey+".yaml"))
	if err != nil {
		return BindingTemplate{}, fmt.Errorf("binding template %s (%s -> %s) not found: %w", owner.String(), from.String(), to.String(), err)
	}

	// Unmarshal the YAML fileContent into a map
	var template BindingTemplate
	if err := yaml.Unmarshal(fileContent, &template); err != nil {
		return BindingTemplate{}, fmt.Errorf("failed to unmarshal yaml: %w", err)
	}

	// Cache the binding template for future use
	cachedBindings[cacheKey] = template

	return template, nil
}

func getConstructTemplateDir(id ConstructTemplateId) (string, error) {
	// trim the klotho package prefix
	parts := strings.SplitN(id.Package, ".", 2)
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid package: %s", id.Package)
	}
	parts = strings.Split(parts[1], ".")

	return strings.ToLower(filepath.Join(append(append([]string{"templates"}, parts...), id.Name)...)), nil
}
