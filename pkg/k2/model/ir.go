package model

import (
	"os"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type ApplicationEnvironment struct {
	SchemaVersion int                  `yaml:"schemaVersion,omitempty"`
	Version       int                  `yaml:"version,omitempty"`
	ProjectURN    string               `yaml:"project_urn,omitempty"`
	AppURN        string               `yaml:"app_urn,omitempty"`
	Environment   string               `yaml:"environment,omitempty"`
	Constructs    map[string]Construct `yaml:"constructs,omitempty"`
}

type Construct struct {
	URN       *URN                   `yaml:"urn,omitempty"`
	Version   int                    `yaml:"version,omitempty"`
	Inputs    map[string]Input       `yaml:"inputs,omitempty"`
	Outputs   map[string]string      `yaml:"outputs,omitempty"`
	Bindings  []Binding              `yaml:"bindings,omitempty"`
	Options   map[string]interface{} `yaml:"options,omitempty"`
	DependsOn []string               `yaml:"dependsOn,omitempty"`
}

type Input struct {
	Type      string      `yaml:"type,omitempty"`
	Value     interface{} `yaml:"value,omitempty"`
	Encrypted bool        `yaml:"encrypted,omitempty"`
	Status    InputStatus `yaml:"status,omitempty"`
	DependsOn []string    `yaml:"dependsOn,omitempty"`
}

type InputStatus string

const (
	Pending  InputStatus = "pending"
	Resolved InputStatus = "resolved"
	Error    InputStatus = "error"
)

type Binding struct {
	URN         string `yaml:"urn,omitempty"`
	BindingType string `yaml:"binding_type,omitempty"`
}

func ReadIRFile(filename string) (*ApplicationEnvironment, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return &ApplicationEnvironment{}, err
	}
	return ParseIRFile(data)
}

func ParseIRFile(content []byte) (*ApplicationEnvironment, error) {
	var appEnv *ApplicationEnvironment
	err := yaml.Unmarshal(content, &appEnv)
	if err != nil {
		zap.S().Errorf("Error unmarshalling IR file: %s", err)
		return &ApplicationEnvironment{}, err
	}

	return appEnv, nil
}
