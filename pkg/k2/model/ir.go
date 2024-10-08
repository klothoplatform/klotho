package model

import (
	"os"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type ApplicationEnvironment struct {
	SchemaVersion int                  `yaml:"schemaVersion,omitempty"`
	Version       int                  `yaml:"version,omitempty"`
	ProjectURN    URN                  `yaml:"project_urn,omitempty"`
	AppURN        URN                  `yaml:"app_urn,omitempty"`
	Environment   string               `yaml:"environment,omitempty"`
	Constructs    map[string]Construct `yaml:"constructs,omitempty"`
	DefaultRegion string               `yaml:"default_region,omitempty"`
}

type Construct struct {
	URN       *URN                   `yaml:"urn,omitempty"`
	Version   int                    `yaml:"version,omitempty"`
	Inputs    map[string]Input       `yaml:"inputs,omitempty"`
	Outputs   map[string]any         `yaml:"outputs,omitempty"`
	Bindings  []Binding              `yaml:"bindings,omitempty"`
	Options   map[string]interface{} `yaml:"options,omitempty"`
	DependsOn []*URN                 `yaml:"dependsOn,omitempty"`
}

type Input struct {
	Value     interface{} `yaml:"value,omitempty"`
	Encrypted bool        `yaml:"encrypted,omitempty"`
	Status    InputStatus `yaml:"status,omitempty"`
	DependsOn string      `yaml:"dependsOn,omitempty"`
}

type InputStatus string

const (
	InputStatusPending  InputStatus = "pending"
	InputStatusResolved InputStatus = "resolved"
	InputStatusError    InputStatus = "error"
)

type Binding struct {
	URN    *URN             `yaml:"urn,omitempty"`
	Inputs map[string]Input `yaml:"inputs,omitempty"`
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
