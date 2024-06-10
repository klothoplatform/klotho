package constructs

import (
	"fmt"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"strings"
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

func (c *ConstructId) FromURN(urn *model.URN) error {
	if urn.Type != "construct" {
		return fmt.Errorf("invalid URN type: %s", urn.Type)
	}

	parts := strings.Split(urn.Subtype, ".")
	packageName := strings.Join(parts[:len(parts)-1], ".")
	constructType := parts[len(parts)-1]

	if packageName == "" || constructType == "" {
		return fmt.Errorf("invalid URN subtype: %s", urn.Subtype)
	}

	c.TemplateId = ConstructTemplateId{
		Package: packageName,
		Name:    constructType,
	}

	c.InstanceId = urn.ResourceID
	return nil
}
