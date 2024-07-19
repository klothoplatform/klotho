package property

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/klothoplatform/klotho/pkg/k2/model"
	"gopkg.in/yaml.v3"
)

type ConstructReference struct {
	URN  model.URN `yaml:"urn" json:"urn"`
	Path string    `yaml:"path" json:"path"`
}

type ConstructType struct {
	Package string `yaml:"package"`
	Name    string `yaml:"name"`
}

var constructTypeRegexp = regexp.MustCompile(`^(?:([\w-]+)\.)+([\w-]+)$`)

func (c *ConstructType) UnmarshalYAML(value *yaml.Node) error {
	var typeString string
	err := value.Decode(&typeString)
	if err != nil {
		return fmt.Errorf("failed to decode construct type: %w", err)
	}

	if !constructTypeRegexp.MatchString(typeString) {
		return fmt.Errorf("invalid construct type: %s", typeString)
	}

	lastDot := strings.LastIndex(typeString, ".")
	c.Name = typeString[lastDot+1:]
	c.Package = typeString[:lastDot]

	return nil
}

func (c *ConstructType) String() string {
	return fmt.Sprintf("%s.%s", c.Package, c.Name)
}

func (c *ConstructType) FromString(id string) error {
	parts := strings.Split(id, ".")
	if len(parts) < 2 {
		return fmt.Errorf("invalid construct template id: %s", id)
	}
	c.Package = strings.Join(parts[:len(parts)-1], ".")
	c.Name = parts[len(parts)-1]
	return nil
}

func ParseConstructType(id string) (ConstructType, error) {
	var c ConstructType
	err := c.FromString(id)
	return c, err
}

func (c *ConstructType) FromURN(urn model.URN) error {
	if urn.Type != "construct" {
		return fmt.Errorf("invalid urn type: %s", urn.Type)
	}
	return c.FromString(urn.Subtype)
}
