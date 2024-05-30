package model

import (
	"gopkg.in/yaml.v3"
	"os"

	"github.com/google/uuid"
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
	Type        ConstructType          `yaml:"type,omitempty"`
	URN         URN                    `yaml:"urn,omitempty"`
	Version     int                    `yaml:"version,omitempty"`
	PulumiStack UUID                   `yaml:"pulumi_stack,omitempty"`
	Status      ConstructStatus        `yaml:"status,omitempty"`
	Inputs      map[string]Input       `yaml:"inputs,omitempty"`
	Outputs     map[string]string      `yaml:"outputs,omitempty"`
	Bindings    []Binding              `yaml:"bindings,omitempty"`
	Options     map[string]interface{} `yaml:"options,omitempty"`
	DependsOn   []string               `yaml:"dependsOn,omitempty"`
}

type UUID struct {
	uuid.UUID
}

func (u *UUID) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	parsedUUID, err := uuid.Parse(s)
	if err != nil {
		return err
	}
	*u = UUID{parsedUUID}
	return nil
}

type ConstructType string

const (
	ContainerType ConstructType = "klotho.aws.Container"
	// Add other construct types as needed
)

type ConstructStatus string

const (
	New            ConstructStatus = "new"
	Creating       ConstructStatus = "creating"
	Created        ConstructStatus = "created"
	Updating       ConstructStatus = "updating"
	Updated        ConstructStatus = "updated"
	Destroying     ConstructStatus = "destroying"
	Destroyed      ConstructStatus = "destroyed"
	CreateFailed   ConstructStatus = "create_failed"
	UpdateFailed   ConstructStatus = "update_failed"
	DestroyFailed  ConstructStatus = "destroy_failed"
	UpdatePending  ConstructStatus = "update_pending"
	DestroyPending ConstructStatus = "destroy_pending"
)

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

func ReadIRFile(filename string) (ApplicationEnvironment, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return ApplicationEnvironment{}, err
	}

	var appEnv ApplicationEnvironment
	err = yaml.Unmarshal(data, &appEnv)
	if err != nil {
		return ApplicationEnvironment{}, err
	}

	return appEnv, nil
}
