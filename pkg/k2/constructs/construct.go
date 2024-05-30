package constructs

import (
	"fmt"
	"github.com/klothoplatform/klotho/pkg/construct"
)

type (
	Construct struct {
		Id        ConstructId          `yaml:"id"`
		Inputs    map[string]any       `yaml:"inputs"`
		Resources map[string]*Resource `yaml:"resources"`
		Edges     []*Edge              `yaml:"edges"`
		Outputs   map[string]any       `yaml:"outputs"`
	}

	ConstructId struct {
		TemplateId ConstructTemplateId `yaml:"template_id"`
		InstanceId string              `yaml:"instance_id"`
	}

	Resource struct {
		Id         construct.ResourceId `yaml:"id"`
		Properties map[string]any       `yaml:"properties"`
	}

	Edge struct {
		From ResourceRef    `yaml:"from"`
		To   ResourceRef    `yaml:"to"`
		Data map[string]any `yaml:"data"`
	}
)

func (e *Edge) PrettyPrint() string {
	return e.From.String() + " -> " + e.To.String()
}

func (e *Edge) String() string {
	return e.PrettyPrint() + " :: " + fmt.Sprintf("%v", e.Data)
}
