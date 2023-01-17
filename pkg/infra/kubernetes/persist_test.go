package kubernetes

import (
	"strings"
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/lang/yaml"
	"github.com/stretchr/testify/assert"
)

func Test_handlePersistForExecUnit(t *testing.T) {
	type testResult struct {
		values []Value
		file   string
	}
	tests := []struct {
		name           string
		unit           HelmExecUnit
		deploymentYaml string
		podYaml        string
		deps           []core.Dependency
		want           testResult
		wantErr        bool
	}{
		{
			name: "orm dependency and deployment",
			unit: HelmExecUnit{
				Name:      "unit",
				Namespace: "default",
			},
			deploymentYaml: `apiVersion: apps/v1
kind: Deployment
metadata:
  creationTimestamp: null
  labels:
    execUnit: testUnit
  name: nginx-deployment
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
      execUnit: testUnit
  strategy: {}
  template:
    metadata:
      creationTimestamp: null
      labels:
        app: nginx
        execUnit: testUnit
    spec:
      containers:
      - image: '{{ .Values.testUnitImage }}'
        name: nginx
        resources: {}
      serviceAccountName: testUnit
status: {}
`,
			deps: []core.Dependency{
				{
					Source: core.ResourceKey{Kind: core.ExecutionUnitKind, Name: "unit"},
					Target: core.ResourceKey{Kind: string(core.PersistORMKind), Name: "unit"},
				},
			},
			want: testResult{
				file: `apiVersion: apps/v1
kind: Deployment
metadata:
  creationTimestamp: null
  labels:
    execUnit: testUnit
  name: nginx-deployment
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
      execUnit: testUnit
  strategy: {}
  template:
    metadata:
      creationTimestamp: null
      labels:
        app: nginx
        execUnit: testUnit
    spec:
      containers:
      - env:
        - name: UNIT_PERSIST_ORM_CONNECTION
          value: '{{ .Values.UNITPERSISTORMCONNECTION }}'
        image: '{{ .Values.testUnitImage }}'
        name: nginx
        resources: {}
      serviceAccountName: testUnit
status: {}
`,
				values: []Value{
					{
						ExecUnitName:        "unit",
						Kind:                "Deployment",
						Type:                string(EnvironmentVariableTransformation),
						Key:                 "UNITPERSISTORMCONNECTION",
						EnvironmentVariable: core.EnvironmentVariable{Name: "UNIT_PERSIST_ORM_CONNECTION", Kind: "persist_orm", ResourceID: "unit", Value: "connection_string"},
					},
				},
			},
		},
		{
			name: "redis node dependency and pod",
			unit: HelmExecUnit{
				Name:      "unit",
				Namespace: "default",
			},
			podYaml: `apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  containers:
  - name: web
    image: nginx`,
			deps: []core.Dependency{
				{
					Source: core.ResourceKey{Kind: core.ExecutionUnitKind, Name: "unit"},
					Target: core.ResourceKey{Kind: string(core.PersistRedisNodeKind), Name: "unit"},
				},
			},
			want: testResult{
				file: `apiVersion: v1
kind: Pod
metadata:
  creationTimestamp: null
  name: test
spec:
  containers:
  - env:
    - name: UNIT_PERSIST_REDIS_HOST
      value: '{{ .Values.UNITPERSISTREDISHOST }}'
    - name: UNIT_PERSIST_REDIS_PORT
      value: '{{ .Values.UNITPERSISTREDISPORT }}'
    image: nginx
    name: web
    resources: {}
status: {}
`,
				values: []Value{
					{
						ExecUnitName:        "unit",
						Kind:                "Pod",
						Type:                string(EnvironmentVariableTransformation),
						Key:                 "UNITPERSISTREDISHOST",
						EnvironmentVariable: core.EnvironmentVariable{Name: "UNIT_PERSIST_REDIS_HOST", Kind: "persist_redis_node", ResourceID: "unit", Value: "host"},
					},
					{
						ExecUnitName:        "unit",
						Kind:                "Pod",
						Type:                string(EnvironmentVariableTransformation),
						Key:                 "UNITPERSISTREDISPORT",
						EnvironmentVariable: core.EnvironmentVariable{Name: "UNIT_PERSIST_REDIS_PORT", Kind: "persist_redis_node", ResourceID: "unit", Value: "port"},
					},
				},
			},
		},
		{
			name: "redis cluster dependency and pod",
			unit: HelmExecUnit{
				Name:      "unit",
				Namespace: "default",
			},
			podYaml: `apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  containers:
  - name: web
    image: nginx`,
			deps: []core.Dependency{
				{
					Source: core.ResourceKey{Kind: core.ExecutionUnitKind, Name: "unit"},
					Target: core.ResourceKey{Kind: string(core.PersistRedisClusterKind), Name: "unit"},
				},
			},
			want: testResult{
				file: `apiVersion: v1
kind: Pod
metadata:
  creationTimestamp: null
  name: test
spec:
  containers:
  - env:
    - name: UNIT_PERSIST_REDIS_HOST
      value: '{{ .Values.UNITPERSISTREDISHOST }}'
    - name: UNIT_PERSIST_REDIS_PORT
      value: '{{ .Values.UNITPERSISTREDISPORT }}'
    image: nginx
    name: web
    resources: {}
status: {}
`,
				values: []Value{
					{
						ExecUnitName:        "unit",
						Kind:                "Pod",
						Type:                string(EnvironmentVariableTransformation),
						Key:                 "UNITPERSISTREDISHOST",
						EnvironmentVariable: core.EnvironmentVariable{Name: "UNIT_PERSIST_REDIS_HOST", Kind: "persist_redis_cluster", ResourceID: "unit", Value: "host"},
					},
					{
						ExecUnitName:        "unit",
						Kind:                "Pod",
						Type:                string(EnvironmentVariableTransformation),
						Key:                 "UNITPERSISTREDISPORT",
						EnvironmentVariable: core.EnvironmentVariable{Name: "UNIT_PERSIST_REDIS_PORT", Kind: "persist_redis_cluster", ResourceID: "unit", Value: "port"},
					},
				},
			},
		},
		{
			name: "has dependency but no pod or deployment file",
			unit: HelmExecUnit{
				Name:      "unit",
				Namespace: "default",
			},
			deps: []core.Dependency{
				{
					Source: core.ResourceKey{Kind: core.ExecutionUnitKind, Name: "unit"},
					Target: core.ResourceKey{Kind: string(core.PersistRedisClusterKind), Name: "unit"},
				},
			},
		},
		{
			name: "no deps",
			unit: HelmExecUnit{
				Name:      "unit",
				Namespace: "default",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			deps := &core.Dependencies{}
			for _, dep := range tt.deps {
				deps.Add(dep.Source, dep.Target)
			}

			if tt.deploymentYaml != "" {
				f, err := yaml.NewFile("deployment.yaml", strings.NewReader(tt.deploymentYaml))
				assert.NoError(err)
				tt.unit.Deployment = f
			}
			if tt.podYaml != "" {
				f, err := yaml.NewFile("pod.yaml", strings.NewReader(tt.podYaml))
				assert.NoError(err)
				tt.unit.Pod = f
			}

			values, err := tt.unit.handlePersistForExecUnit(deps)
			if tt.wantErr {
				assert.Error(err)
				return
			} else if !assert.NoError(err) {
				return

			}
			assert.Equal(tt.want.values, values)

			if tt.deploymentYaml != "" {
				assert.Equal(tt.want.file, string(tt.unit.Deployment.Program()))
			}
			if tt.podYaml != "" {
				assert.Equal(tt.want.file, string(tt.unit.Pod.Program()))
			}
		})
	}
}

func Test_generateEnvVarsForPersist(t *testing.T) {
	tests := []struct {
		name     string
		unit     *core.ExecutionUnit
		resource core.CloudResource
		values   []core.EnvironmentVariable
	}{
		{
			name:     "Wrong dependencies",
			unit:     &core.ExecutionUnit{Name: "main", ExecType: "exec_unit"},
			resource: &core.Persist{Name: "file", Kind: core.PersistFileKind},
			values:   []core.EnvironmentVariable{},
		},
		{
			name:     "orm dependency",
			unit:     &core.ExecutionUnit{Name: "main", ExecType: "exec_unit"},
			resource: &core.Persist{Name: "orm", Kind: core.PersistORMKind},
			values:   []core.EnvironmentVariable{{Name: "ORM_PERSIST_ORM_CONNECTION", Kind: "persist_orm", ResourceID: "orm", Value: string(core.CONNECTION_STRING)}},
		},
		{
			name:     "redis node dependency",
			unit:     &core.ExecutionUnit{Name: "main", ExecType: "exec_unit"},
			resource: &core.Persist{Name: "redisNode", Kind: core.PersistRedisNodeKind},
			values: []core.EnvironmentVariable{
				{Name: "REDISNODE_PERSIST_REDIS_HOST", Kind: "persist_redis_node", ResourceID: "redisNode", Value: string(core.HOST)},
				{Name: "REDISNODE_PERSIST_REDIS_PORT", Kind: "persist_redis_node", ResourceID: "redisNode", Value: string(core.PORT)},
			},
		},
		{
			name:     "redis cluster dependency",
			unit:     &core.ExecutionUnit{Name: "main", ExecType: "exec_unit"},
			resource: &core.Persist{Name: "redisCluster", Kind: core.PersistRedisClusterKind},
			values: []core.EnvironmentVariable{
				{Name: "REDISCLUSTER_PERSIST_REDIS_HOST", Kind: "persist_redis_cluster", ResourceID: "redisCluster", Value: string(core.HOST)},
				{Name: "REDISCLUSTER_PERSIST_REDIS_PORT", Kind: "persist_redis_cluster", ResourceID: "redisCluster", Value: string(core.PORT)},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			assert := assert.New(t)

			results := &core.CompilationResult{}
			results.Add(tt.resource)

			deps := &core.Dependencies{}
			deps.Add(tt.unit.Key(), tt.resource.Key())
			envVars := generateEnvVars(deps, tt.unit.Name)

			assert.Equal(tt.values, envVars)
		})
	}
}
