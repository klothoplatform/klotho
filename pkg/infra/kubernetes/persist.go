package kubernetes

import (
	"strings"

	"github.com/klothoplatform/klotho/pkg/core"
)

type PersistEnvVars string

// Constant portion of the env var name for each of the persist types which require connection info
const (
	ORMEnvVar              PersistEnvVars = "_PERSIST_ORM_CONNECTION"
	RedisNodeHostEnvVar    PersistEnvVars = "_PERSIST_REDIS_NODE_HOST"
	RedisNodePortEnvVar    PersistEnvVars = "_PERSIST_REDIS_NODE_PORT"
	RedisClusterHostEnvVar PersistEnvVars = "_PERSIST_REDIS_CLUSTER_HOST"
	RedisClusterPortEnvVar PersistEnvVars = "_PERSIST_REDIS_CLUSTER_PORT"
)

func (unit *HelmExecUnit) handlePersistForExecUnit(result *core.CompilationResult, deps *core.Dependencies) ([]Value, error) {
	var values []Value
	envVars := generateEnvVars(result, deps, unit.Name)

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

func generateEnvVars(result *core.CompilationResult, deps *core.Dependencies, name string) []string {
	envVars := []string{}
	for _, target := range deps.Downstream(core.ResourceKey{Name: name, Kind: core.ExecutionUnitKind}) {
		res := result.Get(target)
		if p, ok := res.(*core.Persist); ok {
			switch p.Kind {
			case core.PersistORMKind:
				envVars = append(envVars, strings.ToUpper(p.Name)+string(ORMEnvVar))
			case core.PersistRedisNodeKind:
				envVars = append(
					envVars,
					strings.ToUpper(p.Name)+string(RedisNodeHostEnvVar),
					strings.ToUpper(p.Name)+string(RedisNodePortEnvVar),
				)
			case core.PersistRedisClusterKind:
				envVars = append(
					envVars,
					strings.ToUpper(p.Name)+string(RedisClusterHostEnvVar),
					strings.ToUpper(p.Name)+string(RedisClusterPortEnvVar),
				)
			}
		}
	}
	return envVars
}
