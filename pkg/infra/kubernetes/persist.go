package kubernetes

import (
	"github.com/klothoplatform/klotho/pkg/core"
)

type PersistEnvVars string

func (unit *HelmExecUnit) handlePersistForExecUnit(deps *core.Dependencies) ([]Value, error) {
	var values []Value
	envVars := generateEnvVars(deps, unit.Name)

	if len(envVars) == 0 {
		return values, nil
	}

	if unit.Deployment != nil {
		v, err := unit.addEnvsVarToDeployment(envVars)
		if err != nil {
			return nil, err
		}

		values = append(values, v...)
	} else if unit.Pod != nil {
		v, err := unit.addEnvVarToPod(envVars)
		if err != nil {
			return nil, err
		}

		values = append(values, v...)
	}

	return values, nil
}

func generateEnvVars(deps *core.Dependencies, name string) core.EnvironmentVariables {
	envVars := core.EnvironmentVariables{}
	for _, target := range deps.Downstream(core.ResourceKey{Name: name, Kind: core.ExecutionUnitKind}) {
		switch target.Kind {
		case string(core.PersistORMKind):
			envVars = append(envVars, core.GenerateOrmConnStringEnvVar(target.Name, target.Kind))
		case string(core.PersistRedisNodeKind):
			envVars = append(
				envVars,
				core.GenerateRedisHostEnvVar(target.Name, target.Kind),
				core.GenerateRedisPortEnvVar(target.Name, target.Kind),
			)
		case string(core.PersistRedisClusterKind):
			envVars = append(
				envVars,
				core.GenerateRedisHostEnvVar(target.Name, target.Kind),
				core.GenerateRedisPortEnvVar(target.Name, target.Kind),
			)
		}
	}
	return envVars
}
