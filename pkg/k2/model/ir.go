package model

import (
	"os"

	"github.com/google/uuid"
	yaml "gopkg.in/yaml.v2"
)

type ApplicationEnvironment struct {
	SchemaVersion string               `yaml:"schemaVersion"`
	Version       int                  `yaml:"version"`
	URN           string               `yaml:"urn"`
	Constructs    map[string]Construct `yaml:"constructs"`
}

type Construct struct {
	Type        ConstructType          `yaml:"type"`
	URN         string                 `yaml:"urn"`
	Version     int                    `yaml:"version"`
	PulumiStack UUID                   `yaml:"pulumi_stack"`
	Status      ConstructStatus        `yaml:"status"`
	Inputs      map[string]Input       `yaml:"inputs"`
	Outputs     map[string]string      `yaml:"outputs"`
	Bindings    []Binding              `yaml:"bindings"`
	Options     map[string]interface{} `yaml:"options"`
	DependsOn   []string               `yaml:"dependsOn"`
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
	New                           ConstructStatus = "new"
	Creating                      ConstructStatus = "creating"
	Created                       ConstructStatus = "created"
	Updating                      ConstructStatus = "updating"
	Updated                       ConstructStatus = "updated"
	Destroying                    ConstructStatus = "destroying"
	Destroyed                     ConstructStatus = "destroyed"
	CreateFailed                  ConstructStatus = "create_failed"
	UpdateFailed                  ConstructStatus = "update_failed"
	DestroyFailed                 ConstructStatus = "destroy_failed"
	UpdatePending                 ConstructStatus = "update_pending"
	DestroyPendingConstructStatus                 = "destroy_pending"
)

type Input struct {
	Type      string      `yaml:"type"`
	Value     interface{} `yaml:"value"`
	Encrypted bool        `yaml:"encrypted"`
	Status    InputStatus `yaml:"status,omitempty"`
	DependsOn []string    `yaml:"dependsOn"`
}

type InputStatus string

const (
	Pending  InputStatus = "pending"
	Resolved InputStatus = "resolved"
	Error    InputStatus = "error"
)

type Binding struct {
	URN         string `yaml:"urn"`
	BindingType string `yaml:"binding_type"`
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
