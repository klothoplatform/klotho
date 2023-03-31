package envvar

import (
	"errors"
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/config"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"go.uber.org/zap"
)

type (
	EnvVarInjection struct {
		Config *config.Application
	}
)

var (
	SupportedKindMappings = map[string]string{
		"orm":           string(core.PersistORMKind),
		"redis_node":    string(core.PersistRedisNodeKind),
		"redis_cluster": string(core.PersistRedisClusterKind),
	}

	SupportedKindValues = map[string][]string{
		string(core.PersistORMKind):          {"connection_string"},
		string(core.PersistRedisClusterKind): {"host", "port"},
		string(core.PersistRedisNodeKind):    {"host", "port"},
	}
)

func (p EnvVarInjection) Name() string { return "EnvVarInjection" }

func (p EnvVarInjection) Transform(result *core.CompilationResult, deps *core.Dependencies) error {
	var errs multierr.Error

	units := core.GetResourcesOfType[*core.ExecutionUnit](result)
	for _, unit := range units {
		for _, f := range unit.Files() {
			log := zap.L().With(logging.FileField(f)).Sugar()
			ast, ok := f.(*core.SourceFile)
			if !ok {
				log.Debug("Skipping non-source file")
				continue
			}

			for _, annot := range ast.Annotations() {
				cap := annot.Capability
				if cap.Name == annotation.PersistCapability {
					if cap.ID == "" {
						errs.Append(core.NewCompilerError(ast, annot, errors.New("'id' is required")))
					}
					directiveResult, err := ParseDirectiveToEnvVars(cap)
					if err != nil {
						errs.Append(err)
						continue
					}
					if directiveResult.kind == "" {
						continue
					}
					err = handlePersist(directiveResult.kind, cap, unit, result, deps)
					if err != nil {
						errs.Append(err)
						continue
					}
					unit.EnvironmentVariables.AddAll(directiveResult.variables)
				}
			}
		}

	}
	return errs.ErrOrNil()
}

func validateValue(kind string, value string) bool {
	for _, v := range SupportedKindValues[kind] {
		if v == value {
			return true
		}
	}
	return false
}

type EnvironmentVariableDirectiveResult struct {
	kind      string
	variables core.EnvironmentVariables
}

func ParseDirectiveToEnvVars(cap *annotation.Capability) (EnvironmentVariableDirectiveResult, error) {
	overallKind := ""
	envVars := cap.Directives.Object(core.EnvironmentVariableDirective)
	foundVars := core.EnvironmentVariables{}
	if envVars == nil {
		return EnvironmentVariableDirectiveResult{}, nil
	}
	for name, v := range envVars {

		v, ok := v.(string)
		if !ok {
			return EnvironmentVariableDirectiveResult{}, errors.New("environment variable directive must have values as strings")
		}
		valueSplit := strings.Split(v, ".")
		if len(valueSplit) != 2 {
			return EnvironmentVariableDirectiveResult{}, errors.New("invalid environment variable directive value")
		}

		kind := valueSplit[0]
		value := valueSplit[1]

		kind, ok = SupportedKindMappings[kind]
		if !ok {
			return EnvironmentVariableDirectiveResult{}, errors.New("invalid value for 'kind' of environment variable value")
		}

		if !validateValue(kind, value) {
			return EnvironmentVariableDirectiveResult{}, fmt.Errorf("value, %s, is not valid for kind, %s", value, kind)
		}

		if overallKind == "" {
			overallKind = kind
		} else if overallKind != kind {
			return EnvironmentVariableDirectiveResult{}, errors.New("cannot have multiple resource kinds in environment variables for single annotation")
		}

		foundVariable := core.NewEnvironmentVariable(name, kind, cap.ID, value)

		foundVars.Add(foundVariable)
	}

	return EnvironmentVariableDirectiveResult{kind: overallKind, variables: foundVars}, nil
}

func handlePersist(kind string, cap *annotation.Capability, unit *core.ExecutionUnit, result *core.CompilationResult, deps *core.Dependencies) error {
	switch kind {
	case string(core.PersistORMKind):
		resource := core.Persist{
			Kind: core.PersistORMKind,
			Name: cap.ID,
		}
		result.Add(&resource)
		deps.Add(unit.Key(), resource.Key())
	case string(core.PersistRedisClusterKind):
		resource := core.Persist{
			Kind: core.PersistRedisClusterKind,
			Name: cap.ID,
		}
		result.Add(&resource)
		deps.Add(unit.Key(), resource.Key())
	case string(core.PersistRedisNodeKind):
		resource := core.Persist{
			Kind: core.PersistRedisNodeKind,
			Name: cap.ID,
		}
		result.Add(&resource)
		deps.Add(unit.Key(), resource.Key())
	default:
		return fmt.Errorf("unsupported 'kind', %s", kind)
	}
	return nil
}
