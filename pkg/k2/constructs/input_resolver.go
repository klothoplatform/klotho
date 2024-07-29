package constructs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/klothoplatform/klotho/pkg/k2/model"
)

// ResolveInput converts a model.Input to a construct.Input and adds it to the inputs map.
// If the value of the input is a URN, it resolves the URN to a construct.
// If the input's status is not "resolved", it returns an error.
func (ce *ConstructEvaluator) ResolveInput(k string, v model.Input, t InputTemplate) (any, error) {
	if v.Status != "" && v.Status != model.InputStatusResolved {
		if ce.DryRun == model.DryRunNone {
			return nil, fmt.Errorf("input '%s' is not resolved", k)
		}
	}
	switch {
	case strings.HasPrefix(t.Type, "Construct("):
		cType := strings.TrimSuffix(strings.TrimPrefix(t.Type, "Construct("), ")")

		iURN, ok := v.Value.(model.URN)
		if !ok {
			urn, err := model.ParseURN(v.DependsOn)
			if err != nil {
				return nil, fmt.Errorf("input '%s' invalid DependsOn construct URN: %w", k, err)
			}
			iURN = *urn
		}

		if iURN.IsResource() && iURN.Type == "construct" && iURN.Subtype == cType {
			ic, ok := ce.Constructs.Get(iURN)

			if !ok {
				return nil, fmt.Errorf("input '%s' could not find construct %s", k, iURN)
			}
			return ic, nil
		} else {
			return nil, fmt.Errorf("input '%s' invalid construct URN: %+v", k, v)
		}
	case t.Type == "path":
		var err error
		pStr, ok := v.Value.(string)
		if !ok {
			return "", fmt.Errorf("input '%s' invalid path type: expected string, got %T", k, v.Value)
		}
		path, err := handlePathInput(pStr)
		if err != nil {
			return nil, fmt.Errorf("input '%s' could not handle path input: %w", k, err)
		}
		return path, nil

	case t.Type == "KeyValueList":
		return handleKeyValueListInput(v.Value, t)
	default:
		return v.Value, nil
	}
}

// handleKeyValueListInput handles converts a map[string]interface{} to list of key-value pairs.
// Key and value field names are configurable in the input template. The default field names are "Key" and "Value".
func handleKeyValueListInput(value any, t InputTemplate) (any, error) {
	if value == nil {
		return nil, nil
	}

	inputMap, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected input to be of type map[string]any, got %T", value)
	}

	var keyValueList []any

	keyField := "Key"
	valueField := "Value"
	if kf, ok := t.Configuration["keyField"]; ok {
		if kfs, ok := kf.(string); ok {
			keyField = kfs
		}
	}

	if vf, ok := t.Configuration["valueField"]; ok {
		if vfs, ok := vf.(string); ok {
			valueField = vfs
		}
	}

	for key, val := range inputMap {
		kvPair := map[string]any{
			keyField:   key,
			valueField: val,
		}
		keyValueList = append(keyValueList, kvPair)
	}

	return keyValueList, nil
}

func handlePathInput(value string) (string, error) {
	if filepath.IsAbs(value) {
		return value, nil
	}

	// handle relative paths
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("could not get working directory")
	}
	return filepath.Join(wd, value), nil
}
