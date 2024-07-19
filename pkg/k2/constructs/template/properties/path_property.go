package properties

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/k2/constructs/template/property"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
)

type (
	PathProperty struct {
		SanitizeTmpl  *property.SanitizeTmpl
		AllowedValues []string
		SharedPropertyFields
		property.PropertyDetails
		RelativeTo string
	}
)

func (p *PathProperty) SetProperty(properties construct.Properties, value any) error {
	strVal, ok := value.(string)
	if !ok {
		return fmt.Errorf("value %v is not a string", value)
	}
	if strVal == "" {
		return properties.SetProperty(p.Path, "")
	}

	path, err := resolvePath(strVal, p.RelativeTo)
	if err != nil {
		return err
	}
	return properties.SetProperty(p.Path, path)
}

func (p *PathProperty) AppendProperty(properties construct.Properties, value any) error {
	return p.SetProperty(properties, value)
}

func (p *PathProperty) RemoveProperty(properties construct.Properties, value any) error {
	propVal, err := properties.GetProperty(p.Path)
	if errors.Is(err, construct.ErrPropertyDoesNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if propVal == nil {
		return nil
	}
	return properties.RemoveProperty(p.Path, nil)
}

func (p *PathProperty) Details() *property.PropertyDetails {
	return &p.PropertyDetails
}

func (p *PathProperty) Clone() property.Property {
	clone := *p
	return &clone
}

func (p *PathProperty) GetDefaultValue(ctx property.ExecutionContext, data any) (any, error) {
	if p.DefaultValue == nil {
		return p.ZeroValue(), nil
	}
	return p.Parse(p.DefaultValue, ctx, data)
}

func (p *PathProperty) Parse(value any, ctx property.ExecutionContext, data any) (any, error) {
	strVal := ""
	switch val := value.(type) {
	case string:
		err := ctx.ExecuteUnmarshal(val, data, &strVal)
		if err != nil {
			return nil, err
		}
		strVal = val
	case int, int32, int64, float32, float64, bool:
		strVal = fmt.Sprintf("%v", val)
	default:
		return nil, fmt.Errorf("could not parse string property: invalid string value %v (%[1]T)", value)
	}
	if strVal == "" {
		return "", nil
	}
	return resolvePath(strVal, p.RelativeTo)
}

func (p *PathProperty) ZeroValue() any {
	return ""
}

func (p *PathProperty) Contains(value any, contains any) bool {
	vString, ok := value.(string)
	if !ok {
		return false
	}
	cString, ok := contains.(string)
	if !ok {
		return false
	}
	return strings.Contains(vString, cString)
}

func (p *PathProperty) Type() string {
	return "string"
}

func (p *PathProperty) Validate(properties construct.Properties, value any) error {
	if value == nil {
		if p.Required {
			return fmt.Errorf(property.ErrRequiredProperty, p.Path)
		}
		return nil
	}
	stringVal, ok := value.(string)
	if !ok {
		return fmt.Errorf("value %v is not a string", value)
	}

	if len(p.AllowedValues) > 0 && !collectionutil.Contains(p.AllowedValues, stringVal) {
		return fmt.Errorf("value %s is not allowed. allowed values are %s", stringVal, p.AllowedValues)
	}

	if p.SanitizeTmpl != nil {
		return p.SanitizeTmpl.Check(stringVal)
	}
	return nil
}

func (p *PathProperty) SubProperties() property.PropertyMap {
	return nil
}

func resolvePath(path string, basePath string) (string, error) {
	// If the path is absolute, return it as is
	if filepath.IsAbs(path) {
		return path, nil
	}

	// Otherwise, make it relative to the base path or the current working directory
	if basePath == "" {
		var err error
		basePath, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("could not get working directory")
		}
	}
	abs, err := filepath.Abs(filepath.Join(basePath, path))
	if err != nil {
		return "", fmt.Errorf("could not resolve path %s: %w", path, err)
	}
	return abs, nil
}
