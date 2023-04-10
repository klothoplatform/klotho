package kubernetes

import (
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/testutil"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	yaml2 "k8s.io/apimachinery/pkg/util/yaml"
	k8s_yaml "sigs.k8s.io/yaml"
	"strings"
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/lang/dockerfile"
	"github.com/klothoplatform/klotho/pkg/lang/yaml"
	"github.com/stretchr/testify/assert"
)

func Test_GenerateRoleArnPlaceholder(t *testing.T) {

	tests := []struct {
		name string
		want string
	}{
		{
			name: "testUnit",
			want: "testUnitRoleArn",
		},
		{
			name: "second",
			want: "secondRoleArn",
		},
		{
			name: "not-clean",
			want: "notcleanRoleArn",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			testUnit := core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: tt.name}}
			placeholder := GenerateRoleArnPlaceholder(testUnit.ID)
			assert.Equal(tt.want, placeholder)
		})
	}
}

func Test_GenerateImagePlaceholder(t *testing.T) {

	tests := []struct {
		name string
		want string
	}{
		{
			name: "testUnit",
			want: "testUnitImage",
		},
		{
			name: "second",
			want: "secondImage",
		},
		{
			name: "not-clean",
			want: "notcleanImage",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			testUnit := core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: tt.name}}
			placeholder := GenerateImagePlaceholder(testUnit.ID)
			assert.Equal(tt.want, placeholder)
		})
	}
}

func Test_GenerateTargetGroupBindingPlaceholder(t *testing.T) {

	tests := []struct {
		name string
		want string
	}{
		{
			name: "testUnit",
			want: "testUnitTargetGroupArn",
		},
		{
			name: "second",
			want: "secondTargetGroupArn",
		},
		{
			name: "not-clean",
			want: "notcleanTargetGroupArn",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			testUnit := core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: tt.name}}
			placeholder := GenerateTargetGroupBindingPlaceholder(testUnit.ID)
			assert.Equal(tt.want, placeholder)
		})
	}
}

func Test_GenerateEnvVarKeyValue(t *testing.T) {

	tests := []struct {
		key   string
		value string
	}{
		{
			key:   "unit_PERSIST_ORM_CONNECTION",
			value: "unitPERSISTORMCONNECTION",
		},
		{
			key:   "unit-two_PERSIST_ORM_CONNECTION",
			value: "unittwoPERSISTORMCONNECTION",
		},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			assert := assert.New(t)
			_, v := GenerateEnvVarKeyValue(tt.key)
			assert.Equal(tt.value, v)
		})
	}
}

func Test_shouldTransformImage(t *testing.T) {

	tests := []struct {
		name      string
		fileUnits map[string]string
		want      bool
	}{
		{
			name: "should transform",
			fileUnits: map[string]string{
				"Dockerfile": ``,
			},
			want: true,
		},
		{
			name: "should not transform",
			fileUnits: map[string]string{
				"file.js": ``,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			testUnit := core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: tt.name}}

			for path, file := range tt.fileUnits {
				if strings.Contains(path, "Dockerfile") {
					f, err := dockerfile.NewFile(path, strings.NewReader(file))
					if assert.Nil(err) {
						testUnit.Add(f)
					}
				} else {
					f, err := core.NewSourceFile(path, strings.NewReader(file), testLang)
					if assert.Nil(err) {
						testUnit.Add(f)
					}
				}

			}

			transform := shouldTransformImage(&testUnit)
			assert.Equal(tt.want, transform)
		})
	}
}

func Test_shouldTransformServiceAccount(t *testing.T) {

	tests := []struct {
		name      string
		fileUnits map[string]string
		want      bool
	}{
		{
			name: "should transform",
			fileUnits: map[string]string{
				"Dockerfile": ``,
			},
			want: true,
		},
		{
			name: "should not transform",
			fileUnits: map[string]string{
				"file.js": ``,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			testUnit := core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: tt.name}}

			for path, file := range tt.fileUnits {
				if strings.Contains(path, "Dockerfile") {
					f, err := dockerfile.NewFile(path, strings.NewReader(file))
					if assert.Nil(err) {
						testUnit.Add(f)
					}
				} else {
					f, err := core.NewSourceFile(path, strings.NewReader(file), testLang)
					if assert.Nil(err) {
						testUnit.Add(f)
					}
				}

			}

			transform := shouldTransformServiceAccount(&testUnit)
			assert.Equal(tt.want, transform)
		})
	}
}

func Test_transformPod(t *testing.T) {
	type result struct {
		values  []HelmChartValue
		newFile string
	}
	tests := []struct {
		name    string
		file    string
		cfg     config.ExecutionUnit
		want    result
		wantErr bool
	}{
		{
			name: "Basic Pod",
			file: testutil.UnIndent(`
                apiVersion: v1
                kind: Pod
                metadata:
                  name: test
                spec:
                  containers:
                  - name: web
                    image: nginx`),
			want: result{
				values: []HelmChartValue{
					{
						ExecUnitName: "testUnit",
						Kind:         "Pod",
						Type:         string(ImageTransformation),
						Key:          "testUnitImage",
					},
				},
				newFile: testutil.UnIndent(`
                    apiVersion: v1
                    kind: Pod
                    metadata:
                      creationTimestamp: null
                      labels:
                        execUnit: testUnit
                      name: test
                    spec:
                      containers:
                      - image: '{{ .Values.testUnitImage }}'
                        name: web
                        resources: {}
                      serviceAccountName: testUnit
                    status: {}`),
			},
		},
		{
			// This is just to test that we call upsertOnlyContainer.
			// See Test_upsertOnlyContainer for more exhaustive tests.
			name: "container gets upserted",
			file: testutil.UnIndent(`
                apiVersion: v1
                kind: Pod
                metadata:
                  name: test
                spec:
                  containers: []`),
			cfg: config.ExecutionUnit{
				InfraParams: config.InfraParams{
					"limits": map[string]any{
						"cpu": 123,
					},
				},
			},
			want: result{
				values: []HelmChartValue{
					{
						ExecUnitName: "testUnit",
						Kind:         "Pod",
						Type:         string(ImageTransformation),
						Key:          "testUnitImage",
					},
				},
				newFile: testutil.UnIndent(`
                    apiVersion: v1
                    kind: Pod
                    metadata:
                      creationTimestamp: null
                      labels:
                        execUnit: testUnit
                      name: test
                    spec:
                      containers:
                      - image: '{{ .Values.testUnitImage }}'
                        name: testUnit
                        resources:
                          limits:
                            cpu: "123"
                      serviceAccountName: testUnit
                    status: {}`),
			},
		},
		{
			name: "reject Pod with multiple containers",
			file: testutil.UnIndent(`
                apiVersion: v1
                kind: Pod
                metadata:
                  name: test
                spec:
                  containers:
                  - name: web
                    image: nginx
                  - name: web2
                    image: nginx2`),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			testUnit := HelmExecUnit{Name: "testUnit"}

			f, err := yaml.NewFile("pod.yaml", strings.NewReader(tt.file))
			if assert.Nil(err) {
				testUnit.Pod = f
			}

			values, err := testUnit.transformPod(tt.cfg)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.want.values, values)
			assert.Equal(tt.want.newFile, string(testUnit.Pod.Program()))
		})
	}
}

func Test_transformDeployment(t *testing.T) {
	type result struct {
		values      []HelmChartValue
		focusOnPath string
		newFile     string
	}
	basicDeploymentYaml := testutil.UnIndent(`
        apiVersion: apps/v1
        kind: Deployment
        metadata:
          name: nginx-deployment
        spec:
          replicas: 3
          selector:
            matchLabels:
              app: nginx
          template:
            metadata:
              labels:
                app: nginx
            spec:
              containers:
              - name: nginx
                image: nginx:1.14.2`)
	wantValues := []HelmChartValue{
		{
			ExecUnitName: "testUnit",
			Kind:         "Deployment",
			Type:         string(ImageTransformation),
			Key:          "testUnitImage",
		},
	}
	tests := []struct {
		name    string
		file    string
		cfg     config.ExecutionUnit
		want    result
		wantErr bool
	}{
		{
			name: "Basic Deployment",
			file: basicDeploymentYaml,
			want: result{
				values: wantValues,
				newFile: testutil.UnIndent(`
                    apiVersion: apps/v1
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
                    status: {}`),
			},
		},
		{
			// This is just to test that we call upsertOnlyContainer.
			// See Test_upsertOnlyContainer for more exhaustive tests.
			name: "specify cpu int",
			file: basicDeploymentYaml,
			cfg: config.ExecutionUnit{
				InfraParams: config.InfraParams{
					"limits": map[string]any{
						"cpu": 123,
					},
				},
			},
			want: result{
				values:      wantValues,
				focusOnPath: "$.spec.template.spec.containers[0].resources",
				newFile: testutil.UnIndent(`
                    limits:
                        cpu: "123"`),
			},
		},
		{
			name: "no containers specified",
			file: testutil.UnIndent(`
                apiVersion: apps/v1
                kind: Deployment
                metadata:
                  name: nginx-deployment
                spec:
                  replicas: 3
                  selector:
                    matchLabels:
                      app: nginx
                  template:
                    metadata:
                      labels:
                        app: nginx`),
			want: result{
				values: wantValues,
				// note that this adds a container
				newFile: testutil.UnIndent(`
                    apiVersion: apps/v1
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
                            name: testUnit
                            resources: {}
                          serviceAccountName: testUnit
                    status: {}`),
			},
		},
		{
			name: "reject Deployment with multiple containers",
			file: testutil.UnIndent(`
                apiVersion: apps/v1
                kind: Deployment
                metadata:
                  name: nginx-deployment
                spec:
                  replicas: 3
                  selector:
                    matchLabels:
                      app: nginx
                  template:
                    metadata:
                      labels:
                        app: nginx
                    spec:
                      containers:
                      - name: nginx
                        image: nginx:1.14.2
                      - name: nginx2
                        image: nginx:1.14.3`),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			testUnit := HelmExecUnit{Name: "testUnit"}

			f, err := yaml.NewFile("deployment.yaml", strings.NewReader(tt.file))
			if assert.Nil(err) {
				testUnit.Deployment = f
			}

			values, err := testUnit.transformDeployment(tt.cfg)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.want.values, values)
			actualYaml := string(testUnit.Deployment.Program())
			if tt.want.focusOnPath != "" {
				actualYaml = testutil.SafeYamlPath(actualYaml, tt.want.focusOnPath)
			}
			assert.Equal(tt.want.newFile, actualYaml)

			chart := apps.Deployment{}
			err = yaml2.Unmarshal([]byte(actualYaml), &chart)
			assert.NoErrorf(err, "while unmarshalling yaml doc")
		})
	}
}

func Test_addEnvVarToDeployment(t *testing.T) {
	type result struct {
		values  []HelmChartValue
		newFile string
	}
	tests := []struct {
		name    string
		file    string
		envVars core.EnvironmentVariables
		want    result
		wantErr bool
	}{
		{
			name: "Basic Deployment",
			file: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.14.2`,
			envVars: core.EnvironmentVariables{{Name: "SEQUELIZEDB_PERSIST_ORM_CONNECTION"}},
			want: result{
				values: []HelmChartValue{
					{
						ExecUnitName:        "testUnit",
						Kind:                "Deployment",
						Type:                string(EnvironmentVariableTransformation),
						Key:                 "SEQUELIZEDBPERSISTORMCONNECTION",
						EnvironmentVariable: core.NewEnvironmentVariable("SEQUELIZEDB_PERSIST_ORM_CONNECTION", nil, ""),
					},
				},
				newFile: `apiVersion: apps/v1
kind: Deployment
metadata:
  creationTimestamp: null
  name: nginx-deployment
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  strategy: {}
  template:
    metadata:
      creationTimestamp: null
      labels:
        app: nginx
    spec:
      containers:
      - env:
        - name: SEQUELIZEDB_PERSIST_ORM_CONNECTION
          value: '{{ .Values.SEQUELIZEDBPERSISTORMCONNECTION }}'
        image: nginx:1.14.2
        name: nginx
        resources: {}
status: {}
`,
			},
		},
		{
			name: "reject Deployment with multiple containers",
			file: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.14.2
	  - name: nginx2
        image: nginx:1.14.3`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			testUnit := HelmExecUnit{Name: "testUnit"}

			f, err := yaml.NewFile("deployment.yaml", strings.NewReader(tt.file))
			if assert.Nil(err) {
				testUnit.Deployment = f
			}

			values, err := testUnit.addEnvsVarToDeployment(tt.envVars)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.want.values, values)
			assert.Equal(tt.want.newFile, string(testUnit.Deployment.Program()))
		})
	}
}

func Test_addEnvVarToPod(t *testing.T) {
	type result struct {
		values  []HelmChartValue
		newFile string
	}
	tests := []struct {
		name    string
		file    string
		envVars core.EnvironmentVariables
		want    result
		wantErr bool
	}{
		{
			name:    "Basic Pod",
			envVars: core.EnvironmentVariables{{Name: "SEQUELIZEDB_PERSIST_ORM_CONNECTION"}},
			file: `apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  containers:
  - name: web
    image: nginx`,
			want: result{
				values: []HelmChartValue{
					{
						ExecUnitName:        "testUnit",
						Kind:                "Pod",
						Type:                string(EnvironmentVariableTransformation),
						Key:                 "SEQUELIZEDBPERSISTORMCONNECTION",
						EnvironmentVariable: core.NewEnvironmentVariable("SEQUELIZEDB_PERSIST_ORM_CONNECTION", nil, ""),
					},
				},
				newFile: `apiVersion: v1
kind: Pod
metadata:
  creationTimestamp: null
  name: test
spec:
  containers:
  - env:
    - name: SEQUELIZEDB_PERSIST_ORM_CONNECTION
      value: '{{ .Values.SEQUELIZEDBPERSISTORMCONNECTION }}'
    image: nginx
    name: web
    resources: {}
status: {}
`,
			},
		},
		{
			name: "reject Pod with multiple containers",
			file: `apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  containers:
  - name: web
    image: nginx
  - name: web2
    image: nginx2`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			testUnit := HelmExecUnit{Name: "testUnit"}

			f, err := yaml.NewFile("pod.yaml", strings.NewReader(tt.file))
			if assert.Nil(err) {
				testUnit.Pod = f
			}

			values, err := testUnit.addEnvVarToPod(tt.envVars)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.want.values, values)
			assert.Equal(tt.want.newFile, string(testUnit.Pod.Program()))
		})
	}
}

func Test_addUnitsEnvironmentVariables(t *testing.T) {
	type testResult struct {
		values []HelmChartValue
		file   string
	}
	tests := []struct {
		name           string
		unit           HelmExecUnit
		deploymentYaml string
		podYaml        string
		want           testResult
		wantErr        bool
	}{
		{
			name: "unit with deployment",
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
        - name: TESTBUCKET_BUCKET_NAME
          value: '{{ .Values.TESTBUCKETBUCKETNAME }}'
        - name: TESTREDIS_PERSIST_REDIS_HOST
          value: '{{ .Values.TESTREDISPERSISTREDISHOST }}'
        - name: TESTSECRET_CONFIG_SECRET
          value: '{{ .Values.TESTSECRETCONFIGSECRET }}'
        - name: TESTORM_PERSIST_ORM_CONNECTION
          value: '{{ .Values.TESTORMPERSISTORMCONNECTION }}'
        image: '{{ .Values.testUnitImage }}'
        name: nginx
        resources: {}
      serviceAccountName: testUnit
status: {}
`,
				values: []HelmChartValue{
					{
						ExecUnitName:        "unit",
						Kind:                "Deployment",
						Type:                string(EnvironmentVariableTransformation),
						Key:                 "TESTREDISPERSISTREDISHOST",
						EnvironmentVariable: core.GenerateRedisHostEnvVar(&core.RedisCluster{AnnotationKey: core.AnnotationKey{ID: "testRedis"}}),
					},
					{
						ExecUnitName:        "unit",
						Kind:                "Deployment",
						Type:                string(EnvironmentVariableTransformation),
						Key:                 "TESTBUCKETBUCKETNAME",
						EnvironmentVariable: core.GenerateBucketEnvVar(&core.Fs{AnnotationKey: core.AnnotationKey{ID: "testBucket"}}),
					},
					{
						ExecUnitName:        "unit",
						Kind:                "Deployment",
						Type:                string(EnvironmentVariableTransformation),
						Key:                 "TESTORMPERSISTORMCONNECTION",
						EnvironmentVariable: core.GenerateOrmConnStringEnvVar(&core.Orm{AnnotationKey: core.AnnotationKey{ID: "testOrm"}}),
					},
					{
						ExecUnitName:        "unit",
						Kind:                "Deployment",
						Type:                string(EnvironmentVariableTransformation),
						Key:                 "TESTSECRETCONFIGSECRET",
						EnvironmentVariable: core.GenerateSecretEnvVar(&core.Config{AnnotationKey: core.AnnotationKey{ID: "testSecret"}, Secret: true}),
					},
				},
			},
		},
		{
			name: "unit with pod",
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
			want: testResult{
				file: `apiVersion: v1
kind: Pod
metadata:
  creationTimestamp: null
  name: test
spec:
  containers:
  - env:
    - name: TESTBUCKET_BUCKET_NAME
      value: '{{ .Values.TESTBUCKETBUCKETNAME }}'
    - name: TESTREDIS_PERSIST_REDIS_HOST
      value: '{{ .Values.TESTREDISPERSISTREDISHOST }}'
    - name: TESTSECRET_CONFIG_SECRET
      value: '{{ .Values.TESTSECRETCONFIGSECRET }}'
    - name: TESTORM_PERSIST_ORM_CONNECTION
      value: '{{ .Values.TESTORMPERSISTORMCONNECTION }}'
    image: nginx
    name: web
    resources: {}
status: {}
`,
				values: []HelmChartValue{
					{
						ExecUnitName:        "unit",
						Kind:                "Pod",
						Type:                string(EnvironmentVariableTransformation),
						Key:                 "TESTREDISPERSISTREDISHOST",
						EnvironmentVariable: core.GenerateRedisHostEnvVar(&core.RedisCluster{AnnotationKey: core.AnnotationKey{ID: "testRedis"}}),
					},
					{
						ExecUnitName:        "unit",
						Kind:                "Pod",
						Type:                string(EnvironmentVariableTransformation),
						Key:                 "TESTBUCKETBUCKETNAME",
						EnvironmentVariable: core.GenerateBucketEnvVar(&core.Fs{AnnotationKey: core.AnnotationKey{ID: "testBucket"}}),
					},
					{
						ExecUnitName:        "unit",
						Kind:                "Pod",
						Type:                string(EnvironmentVariableTransformation),
						Key:                 "TESTORMPERSISTORMCONNECTION",
						EnvironmentVariable: core.GenerateOrmConnStringEnvVar(&core.Orm{AnnotationKey: core.AnnotationKey{ID: "testOrm"}}),
					},
					{
						ExecUnitName:        "unit",
						Kind:                "Pod",
						Type:                string(EnvironmentVariableTransformation),
						Key:                 "TESTSECRETCONFIGSECRET",
						EnvironmentVariable: core.GenerateSecretEnvVar(&core.Config{AnnotationKey: core.AnnotationKey{ID: "testSecret"}, Secret: true}),
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
			want: testResult{
				values: []HelmChartValue{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			eunit := core.ExecutionUnit{}
			eunit.EnvironmentVariables.Add(core.GenerateBucketEnvVar(&core.Fs{AnnotationKey: core.AnnotationKey{ID: "testBucket"}}))
			eunit.EnvironmentVariables.Add(core.GenerateRedisHostEnvVar(&core.RedisCluster{AnnotationKey: core.AnnotationKey{ID: "testRedis"}}))
			eunit.EnvironmentVariables.Add(core.GenerateSecretEnvVar(&core.Config{AnnotationKey: core.AnnotationKey{ID: "testSecret"}, Secret: true}))
			eunit.EnvironmentVariables.Add(core.GenerateOrmConnStringEnvVar(&core.Orm{AnnotationKey: core.AnnotationKey{ID: "testOrm"}}))

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

			values, err := tt.unit.AddUnitsEnvironmentVariables(&eunit)
			if tt.wantErr {
				assert.Error(err)
				return
			} else if !assert.NoError(err) {
				return

			}
			assert.ElementsMatch(tt.want.values, values)

			if tt.deploymentYaml != "" {
				assert.Equal(tt.want.file, string(tt.unit.Deployment.Program()))
			}
			if tt.podYaml != "" {
				assert.Equal(tt.want.file, string(tt.unit.Pod.Program()))
			}
		})
	}
}

func Test_transformServiceAccount(t *testing.T) {
	type result struct {
		values  []HelmChartValue
		newFile string
	}
	tests := []struct {
		name    string
		file    string
		want    result
		wantErr bool
	}{
		{
			name: "Basic ServiceAccount",
			file: `apiVersion: v1
kind: ServiceAccount
metadata:
  creationTimestamp: null
  name: release-name-nginx-ingress
  namespace: default`,
			want: result{
				values: []HelmChartValue{
					{
						ExecUnitName: "testUnit",
						Kind:         "ServiceAccount",
						Type:         string(ServiceAccountAnnotationTransformation),
						Key:          "testUnitRoleArn",
					},
				},
				newFile: `apiVersion: v1
kind: ServiceAccount
metadata:
  annotations:
    eks.amazonaws.com/role-arn: '{{ .Values.testUnitRoleArn }}'
  creationTimestamp: null
  labels:
    execUnit: testUnit
  name: release-name-nginx-ingress
  namespace: default
`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			testUnit := HelmExecUnit{Name: "testUnit"}

			f, err := yaml.NewFile("pod.yaml", strings.NewReader(tt.file))
			if assert.Nil(err) {
				testUnit.ServiceAccount = f
			}

			values, err := testUnit.transformServiceAccount()
			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.want.values, values)
			assert.Equal(tt.want.newFile, string(testUnit.ServiceAccount.Program()))
		})
	}
}

func Test_transformTargetGroupBinding(t *testing.T) {
	type result struct {
		values  []HelmChartValue
		newFile string
	}
	tests := []struct {
		name    string
		file    string
		want    result
		wantErr bool
	}{
		{
			name: "happy path test",
			file: `apiVersion: elbv2.k8s.aws/v1beta1
kind: TargetGroupBinding
spec:
  serviceRef:
    name: testUnit
    port: 80
  targetGroupARN: REPLACE_ME
`, want: result{
				values: []HelmChartValue{
					{
						ExecUnitName: "testUnit",
						Kind:         "TargetGroupBinding",
						Type:         string(TargetGroupTransformation),
						Key:          "testUnitTargetGroupArn",
					},
				},
				newFile: `apiVersion: elbv2.k8s.aws/v1beta1
kind: TargetGroupBinding
metadata:
  creationTimestamp: null
  labels:
    execUnit: testUnit
spec:
  serviceRef:
    name: testUnit
    port: 80
  targetGroupARN: '{{ .Values.testUnitTargetGroupArn }}'
status: {}
`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			testUnit := HelmExecUnit{Name: "testUnit"}

			f, err := yaml.NewFile("pod.yaml", strings.NewReader(tt.file))
			if assert.Nil(err) {
				testUnit.TargetGroupBinding = f
			}

			values, err := testUnit.transformTargetGroupBinding()
			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.want.values, values)
			assert.Equal(tt.want.newFile, string(testUnit.TargetGroupBinding.Program()))
		})
	}
}

func Test_transformService(t *testing.T) {
	type result struct {
		values  []HelmChartValue
		newFile string
	}
	tests := []struct {
		name    string
		file    string
		want    result
		wantErr bool
	}{
		{
			name: "happy path test",
			file: `apiVersion: v1
kind: Service
metadata:
  name: testUnit
  namespace: default
spec:
  ports:
  - port: 80
    protocol: TCP
    targetPort: 3000
  selector:
    random: something
  sessionAffinity: None
  type: ClusterIP
`,
			want: result{
				newFile: `apiVersion: v1
kind: Service
metadata:
  creationTimestamp: null
  labels:
    execUnit: testUnit
  name: testUnit
  namespace: default
spec:
  ports:
  - port: 80
    protocol: TCP
    targetPort: 3000
  selector:
    execUnit: testUnit
    random: something
  sessionAffinity: None
  type: ClusterIP
status:
  loadBalancer: {}
`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			testUnit := HelmExecUnit{Name: "testUnit"}

			f, err := yaml.NewFile("pod.yaml", strings.NewReader(tt.file))
			if assert.Nil(err) {
				testUnit.Service = f
			}

			values, err := testUnit.transformService()
			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.want.values, values)
			assert.Equal(tt.want.newFile, string(testUnit.Service.Program()))
		})
	}
}

func Test_getServiceAccountName(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		want    string
		wantErr bool
	}{
		{
			name: "name from file",
			file: `apiVersion: v1
kind: ServiceAccount
metadata:
  creationTimestamp: null
  name: pick-me-up
  namespace: default`,
			want: "pick-me-up",
		},
		{
			name: "no file",
			want: "testUnit",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			testUnit := HelmExecUnit{Name: "testUnit"}
			if tt.file != "" {
				f, err := yaml.NewFile("pod.yaml", strings.NewReader(tt.file))
				if assert.Nil(err) {
					testUnit.ServiceAccount = f
				}
			}
			name := testUnit.getServiceAccountName()
			assert.Equal(tt.want, name)
		})
	}
}

func Test_getServiceName(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		want    string
		wantErr bool
	}{
		{
			name: "name from file",
			file: `apiVersion: v1
kind: Service
metadata:
  name: pick-me-up
  namespace: default
spec:
  ports:
  - port: 80
    protocol: TCP
    targetPort: 3000
  selector:
    random: something
  sessionAffinity: None
  type: ClusterIP`,
			want: "pick-me-up",
		},
		{
			name: "name from unit",
			file: `apiVersion: v1
kind: Service
metadata:
  namespace: default
spec:
  ports:
  - port: 80
    protocol: TCP
    targetPort: 3000
  selector:
    random: something
  sessionAffinity: None
  type: ClusterIP`,
			want: "testUnit",
		},
		{
			name: "no file",
			want: "testUnit",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			testUnit := HelmExecUnit{Name: "testUnit"}
			if tt.file != "" {
				f, err := yaml.NewFile("pod.yaml", strings.NewReader(tt.file))
				if assert.Nil(err) {
					testUnit.Service = f
				}
			}
			name := testUnit.getServiceName()
			assert.Equal(tt.want, name)
		})
	}
}

func Test_upsertOnlyContainer(t *testing.T) {
	tests := []struct {
		name        string
		given       []corev1.Container
		wantSuccess bool
	}{
		{
			name:        "nil containers",
			given:       nil,
			wantSuccess: true,
		},
		{
			name:        "empty container",
			given:       []corev1.Container{},
			wantSuccess: true,
		},
		{
			name:        "one container",
			given:       []corev1.Container{corev1.Container{}},
			wantSuccess: true,
		},
		{
			name:        "multiple containers",
			given:       []corev1.Container{corev1.Container{}, corev1.Container{}},
			wantSuccess: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			unit := HelmExecUnit{
				Name: "MyTestHelm",
			}

			// Just a single config, to make sure we call configureContainer.
			// See Test_configureContainer below for more extensive tests
			cfg := config.ExecutionUnit{
				InfraParams: config.InfraParams{
					"limits": map[string]any{
						"cpu": "123",
					},
				},
			}

			valueStr, err := unit.upsertOnlyContainer(&tt.given, cfg)
			if tt.wantSuccess {
				if !assert.NoError(err) {
					return
				}
			} else {
				assert.Error(err)
				return
			}

			if !assert.Equal(1, len(tt.given)) {
				return
			}
			container := tt.given[0]

			// Test the image
			assert.Equal("MyTestHelmImage", valueStr)
			assert.Equal(`{{ .Values.MyTestHelmImage }}`, container.Image)

			// Quick check on config. As mentioned above, we check these more extensively in Test_configureContainer
			cpuQuantity := container.Resources.Limits[corev1.ResourceCPU]
			assert.Equal("123", cpuQuantity.String())
		})
	}
}

func Test_configureContainer(t *testing.T) {
	type result struct {
		focusOnPath   string
		containerYaml string
		err           bool
	}
	tests := []struct {
		name string
		cfg  config.ExecutionUnit
		want result
	}{
		{
			name: "specify cpu str",
			cfg: config.ExecutionUnit{
				InfraParams: config.InfraParams{
					"limits": map[string]any{
						"cpu": "123",
					},
				},
			},
			want: result{
				focusOnPath: "$.resources",
				containerYaml: testutil.UnIndent(`
                    limits:
                        cpu: "123"`),
			},
		},
		{
			name: "specify cpu int",
			cfg: config.ExecutionUnit{
				InfraParams: config.InfraParams{
					"limits": map[string]any{
						"cpu": 123,
					},
				},
			},
			want: result{
				focusOnPath: "$.resources",
				containerYaml: testutil.UnIndent(`
                    limits:
                        cpu: "123"`), // gets converted to str ¯\_(ツ)_/¯
			},
		},
		{
			// From k8s docs:
			// > For CPU resource units, the quantity expression 0.1 is equivalent to the expression 100m, which can be
			// > read as "one hundred millicpu"
			name: "specify cpu float",
			cfg: config.ExecutionUnit{
				InfraParams: config.InfraParams{
					"limits": map[string]any{
						"cpu": 0.1,
					},
				},
			},
			want: result{
				// From k8s docs:
				focusOnPath: "$.resources",
				containerYaml: testutil.UnIndent(`
                    limits:
                        cpu: 100m`), // k8s normalizes it to this
			},
		},
		{
			name: "specify cpu with unit",
			cfg: config.ExecutionUnit{
				InfraParams: config.InfraParams{
					"limits": map[string]any{
						"cpu": "123m",
					},
				},
			},
			want: result{
				focusOnPath: "$.resources",
				containerYaml: testutil.UnIndent(`
                    limits:
                        cpu: 123m`),
			},
		},
		{
			name: "specify cpu with invalid unit",
			cfg: config.ExecutionUnit{
				InfraParams: config.InfraParams{
					"limits": map[string]any{
						"cpu": "123q",
					},
				},
			},
			want: result{
				err: true,
			},
		},
		{
			name: "specify memory with unit",
			cfg: config.ExecutionUnit{
				InfraParams: config.InfraParams{
					"limits": map[string]any{
						"memory": "129M",
					},
				},
			},
			want: result{
				focusOnPath: "$.resources",
				containerYaml: testutil.UnIndent(`
                    limits:
                        memory: 129M`),
			},
		},
		{
			name: "specify both memory and limit",
			cfg: config.ExecutionUnit{
				InfraParams: config.InfraParams{
					"limits": map[string]any{
						"cpu":    123,
						"memory": "129M",
					},
				},
			},
			want: result{
				focusOnPath: "$.resources",
				containerYaml: testutil.UnIndent(`
                    limits:
                        cpu: "123"
                        memory: 129M`),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			unit := HelmExecUnit{}
			container := corev1.Container{}

			err := unit.configureContainer(&container, tt.cfg)
			if tt.want.err {
				assert.Error(err)
				return
			} else if !assert.NoError(err) {
				return
			}

			actualContainerYamlBs, err := k8s_yaml.Marshal(container)
			if !assert.NoError(err) {
				return
			}

			actualContainerYaml := testutil.SafeYamlPath(string(actualContainerYamlBs), tt.want.focusOnPath)

			assert.Equal(tt.want.containerYaml, actualContainerYaml)
		})
	}

}
