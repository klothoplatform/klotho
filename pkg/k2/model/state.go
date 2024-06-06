package model

import (
	"os"
	"sync"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

var stateLock sync.Mutex

type StateManager struct {
	stateFile string
	state     *State
}

type State struct {
	SchemaVersion int                       `yaml:"schemaVersion,omitempty"`
	Version       int                       `yaml:"version,omitempty"`
	ProjectURN    string                    `yaml:"project_urn,omitempty"`
	AppURN        string                    `yaml:"app_urn,omitempty"`
	Environment   string                    `yaml:"environment,omitempty"`
	Constructs    map[string]ConstructState `yaml:"constructs,omitempty"`
}

type ConstructState struct {
	Type        string                 `yaml:"type,omitempty"`
	Status      ConstructStatus        `yaml:"status,omitempty"`
	LastUpdated string                 `yaml:"last_updated,omitempty"`
	Inputs      map[string]Input       `yaml:"inputs,omitempty"`
	Outputs     map[string]string      `yaml:"outputs,omitempty"`
	Bindings    []Binding              `yaml:"bindings,omitempty"`
	Options     map[string]interface{} `yaml:"options,omitempty"`
	DependsOn   []string               `yaml:"dependsOn,omitempty"`
	PulumiStack UUID                   `yaml:"pulumi_stack,omitempty"`
}

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

func NewStateManager(stateFile string) *StateManager {
	return &StateManager{
		stateFile: stateFile,
		state: &State{
			SchemaVersion: 1,
			Version:       1,
			Constructs:    make(map[string]ConstructState),
		},
	}
}

func (sm *StateManager) CheckStateFileExists() bool {
	_, err := os.Stat(sm.stateFile)
	return err == nil
}

func (sm *StateManager) InitState(ir *ApplicationEnvironment) {
	for urn, construct := range ir.Constructs {
		sm.state.Constructs[urn] = ConstructState{
			Type:        string(construct.Type),
			Status:      New,
			LastUpdated: "", // Initial last updated time could be set here
			Inputs:      construct.Inputs,
			Outputs:     construct.Outputs,
			Bindings:    construct.Bindings,
			Options:     construct.Options,
			DependsOn:   construct.DependsOn,
		}
	}
	sm.state.ProjectURN = ir.ProjectURN
	sm.state.AppURN = ir.AppURN
	sm.state.Environment = ir.Environment
}

func (sm *StateManager) LoadState() error {
	data, err := os.ReadFile(sm.stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			sm.state = &State{}
			return nil
		}
		return err
	}
	return yaml.Unmarshal(data, sm.state)
}

func (sm *StateManager) SaveState() error {
	stateLock.Lock()
	defer stateLock.Unlock()

	data, err := yaml.Marshal(sm.state)
	if err != nil {
		return err
	}
	return os.WriteFile(sm.stateFile, data, 0644)
}

func (sm *StateManager) GetState() *State {
	return sm.state
}

func (sm *StateManager) UpdateResourceState(name string, status ConstructStatus, lastUpdated string) {
	if sm.state.Constructs == nil {
		sm.state.Constructs = make(map[string]ConstructState)
	}

	if construct, exists := sm.state.Constructs[name]; exists {
		construct.Status = status
		construct.LastUpdated = lastUpdated
		sm.state.Constructs[name] = construct
	} else {
		sm.state.Constructs[name] = ConstructState{
			Status:      status,
			LastUpdated: lastUpdated,
		}
	}
}
