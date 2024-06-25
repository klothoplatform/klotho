package model

import (
	"fmt"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type StateManager struct {
	stateFile string
	state     *State
	mutex     sync.Mutex
}

type State struct {
	SchemaVersion int                       `yaml:"schemaVersion,omitempty"`
	Version       int                       `yaml:"version,omitempty"`
	ProjectURN    string                    `yaml:"project_urn,omitempty"`
	AppURN        string                    `yaml:"app_urn,omitempty"`
	Environment   string                    `yaml:"environment,omitempty"`
	DefaultRegion string                    `yaml:"default_region,omitempty"`
	Constructs    map[string]ConstructState `yaml:"constructs,omitempty"`
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
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	for urn, construct := range ir.Constructs {
		sm.state.Constructs[urn] = ConstructState{
			Status:      ConstructPending,
			LastUpdated: time.Now().Format(time.RFC3339),
			Inputs:      construct.Inputs,
			Outputs:     construct.Outputs,
			Bindings:    construct.Bindings,
			Options:     construct.Options,
			DependsOn:   construct.DependsOn,
			URN:         construct.URN,
		}
	}
	sm.state.ProjectURN = ir.ProjectURN
	sm.state.AppURN = ir.AppURN
	sm.state.Environment = ir.Environment
	sm.state.DefaultRegion = ir.DefaultRegion
}

func (sm *StateManager) LoadState() error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

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
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	data, err := yaml.Marshal(sm.state)
	if err != nil {
		return err
	}
	return os.WriteFile(sm.stateFile, data, 0644)
}

func (sm *StateManager) GetState() *State {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	return sm.state
}

func (sm *StateManager) UpdateResourceState(name string, status ConstructStatus, lastUpdated string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if sm.state.Constructs == nil {
		sm.state.Constructs = make(map[string]ConstructState)
	}

	construct, exists := sm.state.Constructs[name]
	if !exists {
		return fmt.Errorf("construct %s not found", name)
	}

	if !isValidTransition(construct.Status, status) {
		return fmt.Errorf("invalid transition from %s to %s", construct.Status, status)
	}

	construct.Status = status
	construct.LastUpdated = lastUpdated
	sm.state.Constructs[name] = construct

	return nil
}

func (sm *StateManager) GetConstructState(name string) (ConstructState, bool) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	construct, exists := sm.state.Constructs[name]
	return construct, exists
}

func (sm *StateManager) SetConstructState(construct ConstructState) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.state.Constructs[construct.URN.ResourceID] = construct
}

func (sm *StateManager) GetAllConstructs() map[string]ConstructState {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	return sm.state.Constructs
}

func (sm *StateManager) TransitionConstructState(construct *ConstructState, nextStatus ConstructStatus) error {
	if isValidTransition(construct.Status, nextStatus) {
		zap.L().Debug("Transitioning construct", zap.String("urn", construct.URN.String()), zap.String("from", string(construct.Status)), zap.String("to", string(nextStatus)))
		construct.Status = nextStatus
		construct.LastUpdated = time.Now().Format(time.RFC3339)
		sm.SetConstructState(*construct)
		return nil
	}
	return fmt.Errorf("invalid state transition from %s to %s", construct.Status, nextStatus)
}

// RegisterOutputValues registers the resolved output values of a construct in the state manager and resolves any inputs that depend on the provided outputs
func (sm *StateManager) RegisterOutputValues(urn URN, outputs map[string]any) error {
	if sm.state.Constructs == nil {
		return fmt.Errorf("%s not found in state", urn.String())
	}

	if construct, exists := sm.state.Constructs[urn.ResourceID]; exists {
		if construct.Outputs == nil {
			construct.Outputs = make(map[string]any)
		}

		for k, v := range outputs {
			if construct.Outputs == nil {
				construct.Outputs = make(map[string]any)
			}
			construct.Outputs[k] = v
		}
		sm.state.Constructs[urn.ResourceID] = construct
	} else {
		return fmt.Errorf("%s not found in state", urn.String())
	}

	// Resolve inputs that depend on the outputs of this construct or that directly reference this construct
	for _, c := range sm.state.Constructs {
		if urn.Equals(c.URN) {
			continue // Skip the construct that provided the outputs
		}

		hasDep := false
		// Skip constructs that don't depend on this construct
		for _, dep := range c.DependsOn {
			if dep.Equals(urn) {
				hasDep = true
				break
			}
		}
		if !hasDep {
			continue
		}

		for k, input := range c.Inputs {
			if input.DependsOn == urn.String() {
				input.Status = InputStatusResolved
				input.Value = urn
				c.Inputs[k] = input
			}

			if o, ok := outputs[input.DependsOn]; ok {
				input.Value = o
				input.Status = InputStatusResolved
				c.Inputs[k] = input
			}
		}

	}

	return nil
}
