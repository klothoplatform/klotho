package engine

import (
	"os"

	"github.com/klothoplatform/klotho/pkg/core"

	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

func (e *Engine) LoadConstructGraphFromFile(path string) error {

	input := core.InputGraph{}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close() // nolint:errcheck

	err = yaml.NewDecoder(f).Decode(&input)
	if err != nil {
		return err
	}

	err = core.LoadConstructsIntoGraph(input, e.Context.InitialState)
	if err != nil {
		return errors.Errorf("Error Loading graph for constructs %s", err.Error())
	}

	err = e.Provider.LoadGraph(input, e.Context.InitialState)
	if err != nil {
		return errors.Errorf("Error Loading graph for provider %s. %s", e.Provider.Name(), err.Error())
	}

	for _, metadata := range input.ResourceMetadata {
		resource := e.Context.InitialState.GetConstruct(metadata.Id)
		md, err := yaml.Marshal(metadata.Metadata)
		if err != nil {
			return err
		}
		err = yaml.Unmarshal(md, resource)
		if err != nil {
			return err
		}
	}

	return nil
}

func (e *Engine) LoadConstraintsFromFile(path string) (map[constraints.ConstraintScope][]constraints.Constraint, error) {

	type Input struct {
		Constraints []any             `yaml:"constraints"`
		Resources   []core.ResourceId `yaml:"resources"`
		Edges       []core.OutputEdge `yaml:"edges"`
	}

	input := Input{}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close() // nolint:errcheck

	err = yaml.NewDecoder(f).Decode(&input)
	if err != nil {
		return nil, err
	}

	bytesArr, err := yaml.Marshal(input.Constraints)
	if err != nil {
		return nil, err
	}
	return constraints.ParseConstraintsFromFile(bytesArr)
}
