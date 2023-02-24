package kubernetes

import (
	"strings"
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/lang/dockerfile"
	yamlLang "github.com/klothoplatform/klotho/pkg/lang/yaml"
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
			testUnit := core.ExecutionUnit{Name: tt.name}
			placeholder := GenerateRoleArnPlaceholder(testUnit.Name)
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
			testUnit := core.ExecutionUnit{Name: tt.name}
			placeholder := GenerateImagePlaceholder(testUnit.Name)
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
			testUnit := core.ExecutionUnit{Name: tt.name}
			placeholder := GenerateTargetGroupBindingPlaceholder(testUnit.Name)
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
			testUnit := core.ExecutionUnit{Name: tt.name}

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
			testUnit := core.ExecutionUnit{Name: tt.name}

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
		values  []Value
		newFile string
	}
	tests := []struct {
		name    string
		file    string
		want    result
		wantErr bool
	}{
		{
			name: "Basic Pod",
			file: `apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  containers:
  - name: web
    image: nginx`,
			want: result{
				values: []Value{
					{
						ExecUnitName: "testUnit",
						Kind:         "Pod",
						Type:         string(ImageTransformation),
						Key:          "testUnitImage",
					},
				},
				newFile: `apiVersion: v1
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

			f, err := yamlLang.NewFile("pod.yaml", strings.NewReader(tt.file))
			if assert.Nil(err) {
				testUnit.Pod = f
			}

			values, err := testUnit.transformPod()
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
		values  []Value
		newFile string
	}
	tests := []struct {
		name    string
		file    string
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
			want: result{
				values: []Value{
					{
						ExecUnitName: "testUnit",
						Kind:         "Deployment",
						Type:         string(ImageTransformation),
						Key:          "testUnitImage",
					},
				},
				newFile: `apiVersion: apps/v1
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

			f, err := yamlLang.NewFile("deployment.yaml", strings.NewReader(tt.file))
			if assert.Nil(err) {
				testUnit.Deployment = f
			}

			values, err := testUnit.transformDeployment()
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

func Test_addEnvVarToDeployment(t *testing.T) {
	type result struct {
		values  []Value
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
				values: []Value{
					{
						ExecUnitName:        "testUnit",
						Kind:                "Deployment",
						Type:                string(EnvironmentVariableTransformation),
						Key:                 "SEQUELIZEDBPERSISTORMCONNECTION",
						EnvironmentVariable: core.EnvironmentVariable{Name: "SEQUELIZEDB_PERSIST_ORM_CONNECTION"},
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

			f, err := yamlLang.NewFile("deployment.yaml", strings.NewReader(tt.file))
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
		values  []Value
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
				values: []Value{
					{
						ExecUnitName:        "testUnit",
						Kind:                "Pod",
						Type:                string(EnvironmentVariableTransformation),
						Key:                 "SEQUELIZEDBPERSISTORMCONNECTION",
						EnvironmentVariable: core.EnvironmentVariable{Name: "SEQUELIZEDB_PERSIST_ORM_CONNECTION"},
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

			f, err := yamlLang.NewFile("pod.yaml", strings.NewReader(tt.file))
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

func Test_transformServiceAccount(t *testing.T) {
	type result struct {
		values  []Value
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
				values: []Value{
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

			f, err := yamlLang.NewFile("pod.yaml", strings.NewReader(tt.file))
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
		values  []Value
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
				values: []Value{
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

			f, err := yamlLang.NewFile("pod.yaml", strings.NewReader(tt.file))
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
		values  []Value
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

			f, err := yamlLang.NewFile("pod.yaml", strings.NewReader(tt.file))
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
				f, err := yamlLang.NewFile("pod.yaml", strings.NewReader(tt.file))
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
				f, err := yamlLang.NewFile("pod.yaml", strings.NewReader(tt.file))
				if assert.Nil(err) {
					testUnit.Service = f
				}
			}
			name := testUnit.getServiceName()
			assert.Equal(tt.want, name)
		})
	}
}
