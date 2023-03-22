package kubernetes

import (
	"bytes"
	"strings"
	"testing"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/graph"
	"github.com/klothoplatform/klotho/pkg/lang/dockerfile"
	yamlLang "github.com/klothoplatform/klotho/pkg/lang/yaml"
	"github.com/stretchr/testify/assert"
)

func Test_AssignFilesToUnits(t *testing.T) {
	type TestUnit struct {
		name               string
		podPath            string
		deploymentPath     string
		serviceAccountPath string
		servicePath        string
	}
	tests := []struct {
		name      string
		fileUnits map[string]string
		chart     KlothoHelmChart
		want      []TestUnit
		wantErr   bool
	}{
		{
			name: "Basic Pod",
			chart: KlothoHelmChart{
				ExecutionUnits: []*HelmExecUnit{
					{Name: "unit1"},
				},
			},
			fileUnits: map[string]string{"pod.yaml": `apiVersion: v1
kind: Pod
spec:
  containers:
  - name: web
    image: nginx`},
			want: []TestUnit{
				{
					name:    "unit1",
					podPath: "pod.yaml",
				},
			},
		},
		{
			name: "Basic Deployment",
			chart: KlothoHelmChart{
				ExecutionUnits: []*HelmExecUnit{
					{Name: "unit1"},
				},
			},
			fileUnits: map[string]string{"deployment.yaml": `apiVersion: apps/v1
kind: Deployment
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
			},
			want: []TestUnit{
				{
					name:           "unit1",
					deploymentPath: "deployment.yaml",
				},
			},
		},
		{
			name: "Basic ServiceAccount",
			chart: KlothoHelmChart{
				ExecutionUnits: []*HelmExecUnit{
					{Name: "unit1"},
				},
			},
			fileUnits: map[string]string{"ServiceAccount.yaml": `apiVersion: v1
kind: ServiceAccount
metadata:
  name: release-name-nginx-ingress
  namespace: default`,
			},
			want: []TestUnit{
				{
					name:               "unit1",
					serviceAccountPath: "ServiceAccount.yaml",
				},
			},
		},
		{
			name: "Basic Service",
			chart: KlothoHelmChart{
				ExecutionUnits: []*HelmExecUnit{
					{Name: "unit1"},
				},
			},
			fileUnits: map[string]string{"Service.yaml": `apiVersion: v1
kind: Service
spec:
  ports:
  - port: 80
    protocol: TCP
    targetPort: 3000
  selector:
    execUnit: name`,
			},
			want: []TestUnit{
				{
					name:        "unit1",
					servicePath: "Service.yaml",
				},
			},
		},
		{
			name: "Multi unit Pod",
			chart: KlothoHelmChart{
				ExecutionUnits: []*HelmExecUnit{
					{Name: "unit1"},
					{Name: "unit2"},
				},
			},
			fileUnits: map[string]string{"pod.yaml": `apiVersion: v1
kind: Pod
metadata:
  name: unit1
spec:
  containers:
  - name: web
    image: nginx`,
				"pod2.yaml": `apiVersion: v1
kind: Pod
metadata:
  name: notunit2
spec:
  containers:
  - name: web
    image: nginx`},
			want: []TestUnit{
				{
					name:    "unit1",
					podPath: "pod.yaml",
				},
				{
					name: "unit2",
				},
			},
		},
		{
			name: "multi unit Deployment",
			chart: KlothoHelmChart{
				ExecutionUnits: []*HelmExecUnit{
					{Name: "unit1"},
					{Name: "unit2"},
				},
			},
			fileUnits: map[string]string{"deployment.yaml": `apiVersion: apps/v1
kind: Deployment
metadata:
  name: unit1
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
				"deployment2.yaml": `apiVersion: apps/v1
kind: Deployment
metadata:
  name: notunit2
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
			},
			want: []TestUnit{
				{
					name:           "unit1",
					deploymentPath: "deployment.yaml",
				},
				{
					name: "unit2",
				},
			},
		},
		{
			name: "multi unit ServiceAccount",
			chart: KlothoHelmChart{
				ExecutionUnits: []*HelmExecUnit{
					{Name: "unit1"},
					{Name: "unit2"},
				},
			},
			fileUnits: map[string]string{"ServiceAccount.yaml": `apiVersion: v1
kind: ServiceAccount
metadata:
  name: unit1
  namespace: default`,
				"ServiceAccount2.yaml": `apiVersion: v1
kind: ServiceAccount
metadata:
  name: notunit2
  namespace: default`,
			},
			want: []TestUnit{
				{
					name:               "unit1",
					serviceAccountPath: "ServiceAccount.yaml",
				},
				{
					name: "unit2",
				},
			},
		},
		{
			name: "multi unit Service",
			chart: KlothoHelmChart{
				ExecutionUnits: []*HelmExecUnit{
					{Name: "unit1"},
					{Name: "unit2"},
				},
			},
			fileUnits: map[string]string{"Service.yaml": `apiVersion: v1
kind: Service
metadata:
  name: unit1
spec:
  ports:
  - port: 80
    protocol: TCP
    targetPort: 3000
  selector:
    execUnit: name`,
				"Service2.yaml": `apiVersion: v1
kind: Service
metadata:
  name: notunit2
spec:
  ports:
  - port: 80
    protocol: TCP
    targetPort: 3000
  selector:
    execUnit: name`,
			},
			want: []TestUnit{
				{
					name:        "unit1",
					servicePath: "Service.yaml",
				},
				{
					name: "unit2",
				},
			},
		},
		{
			name: "single unit pod and deployment error",
			chart: KlothoHelmChart{
				ExecutionUnits: []*HelmExecUnit{
					{Name: "unit1"},
				},
			},
			fileUnits: map[string]string{"pod.yaml": `apiVersion: v1
kind: Pod
spec:
  containers:
  - name: web
    image: nginx`,
				"deployment.yaml": `apiVersion: apps/v1
kind: Deployment
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
			},
			wantErr: true,
		},
		{
			name: "multi unit pod and deployment error",
			chart: KlothoHelmChart{
				ExecutionUnits: []*HelmExecUnit{
					{Name: "unit1"},
					{Name: "unit2"},
				},
			},
			fileUnits: map[string]string{"pod.yaml": `apiVersion: v1
kind: Pod
metadata:
  name: unit1
spec:
  containers:
  - name: web
    image: nginx`,
				"deployment.yaml": `apiVersion: apps/v1
kind: Deployment
metadata:
  name: unit1
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
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			for path, file := range tt.fileUnits {
				f, err := yamlLang.NewFile(path, strings.NewReader(file))
				if assert.Nil(err) {
					tt.chart.Files = append(tt.chart.Files, f)
				}
			}

			err := tt.chart.AssignFilesToUnits()
			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			for _, hu := range tt.want {
				for _, cu := range tt.chart.ExecutionUnits {
					if hu.name == cu.Name {
						if hu.podPath != "" {
							assert.Equal(hu.podPath, cu.Pod.Path())
						} else {
							assert.Nil(cu.Pod)
						}
						if hu.deploymentPath != "" {
							assert.Equal(hu.deploymentPath, cu.Deployment.Path())
						} else {
							assert.Nil(cu.Deployment)
						}
						if hu.serviceAccountPath != "" {
							assert.Equal(hu.serviceAccountPath, cu.ServiceAccount.Path())
						} else {
							assert.Nil(cu.ServiceAccount)
						}
						if hu.servicePath != "" {
							assert.Equal(hu.servicePath, cu.Service.Path())
						} else {
							assert.Nil(cu.Service)
						}
					}
				}
			}
		})
	}
}

func Test_handleExecutionUnit(t *testing.T) {
	tests := []struct {
		name          string
		chart         KlothoHelmChart
		hasDockerfile bool
		cfg           config.ExecutionUnit
		want          []Value
		wantErr       bool
	}{
		{
			name: "no transforms",
			chart: KlothoHelmChart{
				Name: "test",
				ExecutionUnits: []*HelmExecUnit{
					{
						Name:      "unit",
						Namespace: "default",
					},
				},
			},
			hasDockerfile: false,
			cfg:           config.ExecutionUnit{},
			want:          []Value{},
		},
		{
			name: "only dockerfile",
			chart: KlothoHelmChart{
				Name: "test",
				ExecutionUnits: []*HelmExecUnit{
					{
						Name:      "unit",
						Namespace: "default",
					},
				},
			},
			hasDockerfile: true,
			cfg:           config.ExecutionUnit{},
			want: []Value{
				{
					ExecUnitName: "unit",
					Kind:         "Deployment",
					Type:         "image",
					Key:          "unitImage",
				},
				{
					ExecUnitName: "unit",
					Kind:         "ServiceAccount",
					Type:         "service_account_annotation",
					Key:          "unitRoleArn",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			testUnit := tt.chart.ExecutionUnits[0]

			eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "unit", Capability: annotation.ExecutionUnitCapability}}
			if tt.hasDockerfile {
				dockerF, err := dockerfile.NewFile("Dockerfile", bytes.NewBuffer([]byte{}))
				if !assert.NoError(err) {
					return
				}
				eu.Add(dockerF)
			}
			constructGraph := graph.NewDirected[core.Construct]()
			transformations, err := tt.chart.handleExecutionUnit(testUnit, eu, tt.cfg, constructGraph)
			if tt.wantErr {
				assert.Error(err)
				return
			} else if !assert.NoError(err) {
				return

			}
			assert.Equal(tt.want, transformations)
		})
	}
}

func Test_handleUpstreamUnitDependencies(t *testing.T) {
	type testResult struct {
		values []Value
		files  []string
	}
	tests := []struct {
		name    string
		chart   KlothoHelmChart
		deps    []graph.Edge[core.Construct]
		want    testResult
		wantErr bool
	}{
		{
			name: "gateway dep",
			chart: KlothoHelmChart{
				Name: "test",
				ExecutionUnits: []*HelmExecUnit{
					{
						Name:      "unit",
						Namespace: "default",
					},
				},
			},
			deps: []graph.Edge[core.Construct]{
				{
					Source:      &core.Gateway{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExposeCapability}},
					Destination: &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{Capability: annotation.ExecutionUnitCapability, ID: "unit"}},
				},
			},
			want: testResult{
				files: []string{"test/templates/unit-targetgroupbinding.yaml"},
				values: []Value{
					{
						ExecUnitName: "unit",
						Kind:         "TargetGroupBinding",
						Type:         string(TargetGroupTransformation),
						Key:          "unitTargetGroupArn",
					},
				},
			},
		},
		{
			name: "exec unit dep",
			chart: KlothoHelmChart{
				Name: "test",
				ExecutionUnits: []*HelmExecUnit{
					{
						Name:      "unit",
						Namespace: "default",
					},
				},
			},
			deps: []graph.Edge[core.Construct]{
				{
					Source:      &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{Capability: annotation.ExecutionUnitCapability, ID: "test"}},
					Destination: &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{Capability: annotation.ExecutionUnitCapability, ID: "unit"}},
				},
			},
			want: testResult{
				files: []string{"test/templates/unit-serviceexport.yaml"},
			},
		},
		{
			name: "multiple deps",
			chart: KlothoHelmChart{
				Name: "test",
				ExecutionUnits: []*HelmExecUnit{
					{
						Name:      "unit",
						Namespace: "default",
					},
				},
			},
			deps: []graph.Edge[core.Construct]{
				{
					Source:      &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{Capability: annotation.ExecutionUnitCapability, ID: "test"}},
					Destination: &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{Capability: annotation.ExecutionUnitCapability, ID: "unit"}},
				},
				{
					Source:      &core.Gateway{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExposeCapability}},
					Destination: &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{Capability: annotation.ExecutionUnitCapability, ID: "unit"}},
				},
			},
			want: testResult{
				files: []string{"test/templates/unit-serviceexport.yaml", "test/templates/unit-targetgroupbinding.yaml"},
				values: []Value{
					{
						ExecUnitName: "unit",
						Kind:         "TargetGroupBinding",
						Type:         string(TargetGroupTransformation),
						Key:          "unitTargetGroupArn",
					},
				},
			},
		},
		{
			name: "no deps",
			chart: KlothoHelmChart{
				Name: "test",
				ExecutionUnits: []*HelmExecUnit{
					{
						Name:      "unit",
						Namespace: "default",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			testUnit := tt.chart.ExecutionUnits[0]
			constructGraph := graph.NewDirected[core.Construct]()
			for _, dep := range tt.deps {
				constructGraph.AddVertex(dep.Source)
				constructGraph.AddVertex(dep.Destination)
				constructGraph.AddEdge(dep.Source, dep.Destination)
			}
			values, err := tt.chart.handleUpstreamUnitDependencies(testUnit, constructGraph)
			if tt.wantErr {
				assert.Error(err)
				return
			} else if !assert.NoError(err) {
				return

			}
			assert.Equal(tt.want.values, values)
			for _, f := range tt.want.files {
				if strings.Contains(f, "targetgroupbinding") {
					assert.Equal(f, testUnit.TargetGroupBinding.Path())
				}
				if strings.Contains(f, "serviceexport") {
					assert.Equal(f, testUnit.ServiceExport.Path())
				}
			}
		})
	}
}

func Test_addDeployment(t *testing.T) {
	type TestUnit struct {
		deploymentPath string
		deploymentFile string
		values         []Value
	}
	tests := []struct {
		name    string
		chart   KlothoHelmChart
		want    TestUnit
		wantErr bool
	}{
		{
			name: "happy path test",
			chart: KlothoHelmChart{
				Name: "test",
				ExecutionUnits: []*HelmExecUnit{
					{
						Name:      "unit",
						Namespace: "default",
					},
				},
			},
			want: TestUnit{
				deploymentPath: "test/templates/unit-deployment.yaml",
				deploymentFile: `apiVersion: apps/v1
kind: Deployment
metadata:
  creationTimestamp: null
  labels:
    execUnit: unit
  name: unit
  namespace: default
spec:
  replicas: 2
  selector:
    matchLabels:
      execUnit: unit
  strategy:
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 1
    type: RollingUpdate
  template:
    metadata:
      creationTimestamp: null
      labels:
        execUnit: unit
        klotho-fargate-enabled: "false"
    spec:
      containers:
      - image: '{{ .Values.unitImage }}'
        name: unit
        resources: {}
      serviceAccount: unit
      serviceAccountName: unit
status: {}
`,
				values: []Value{
					{
						ExecUnitName: "unit",
						Kind:         "Deployment",
						Type:         "image",
						Key:          "unitImage",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			testUnit := tt.chart.ExecutionUnits[0]
			values, err := tt.chart.addDeployment(testUnit)
			if !assert.NoError(err) {
				return
			}
			assert.Len(tt.chart.Files, 1)
			assert.Equal(tt.want.deploymentPath, testUnit.Deployment.Path())
			assert.Equal(tt.want.deploymentFile, string(testUnit.Deployment.Program()))
			assert.Equal(testUnit.Deployment, tt.chart.Files[0])
			assert.Equal(tt.want.values, values)
		})
	}
}

func Test_addServiceAccount(t *testing.T) {
	type TestUnit struct {
		serviceAccountPath string
		serviceAccountFile string
		values             []Value
	}
	tests := []struct {
		name  string
		chart KlothoHelmChart
		want  TestUnit
	}{
		{
			name: "happy path test",
			chart: KlothoHelmChart{
				Name: "test",
				ExecutionUnits: []*HelmExecUnit{
					{
						Name:      "unit",
						Namespace: "default",
					},
				},
			},
			want: TestUnit{
				serviceAccountPath: "test/templates/unit-serviceaccount.yaml",
				serviceAccountFile: `apiVersion: v1
automountServiceAccountToken: true
kind: ServiceAccount
metadata:
  annotations:
    eks.amazonaws.com/role-arn: '{{ .Values.unitRoleArn }}'
  creationTimestamp: null
  labels:
    execUnit: unit
  name: unit
  namespace: default
`,
				values: []Value{
					{
						ExecUnitName: "unit",
						Kind:         "ServiceAccount",
						Type:         "service_account_annotation",
						Key:          "unitRoleArn",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			testUnit := tt.chart.ExecutionUnits[0]
			values, err := tt.chart.addServiceAccount(testUnit)
			if !assert.NoError(err) {
				return
			}
			assert.Len(tt.chart.Files, 1)
			assert.Equal(tt.want.serviceAccountPath, testUnit.ServiceAccount.Path())
			assert.Equal(tt.want.serviceAccountFile, string(testUnit.ServiceAccount.Program()))
			assert.Equal(testUnit.ServiceAccount, tt.chart.Files[0])
			assert.Equal(tt.want.values, values)
		})
	}
}

func Test_addService(t *testing.T) {
	type TestUnit struct {
		servicePath string
		serviceFile string
		values      []Value
	}
	tests := []struct {
		name  string
		chart KlothoHelmChart
		want  TestUnit
	}{
		{
			name: "happy path test",
			chart: KlothoHelmChart{
				Name: "test",
				ExecutionUnits: []*HelmExecUnit{
					{
						Name:      "unit",
						Namespace: "default",
					},
				},
			},
			want: TestUnit{
				servicePath: "test/templates/unit-service.yaml",
				serviceFile: `apiVersion: v1
kind: Service
metadata:
  creationTimestamp: null
  labels:
    execUnit: unit
  name: unit
  namespace: default
spec:
  ports:
  - port: 80
    protocol: TCP
    targetPort: 3000
  selector:
    execUnit: unit
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
			testUnit := tt.chart.ExecutionUnits[0]
			values, err := tt.chart.addService(testUnit)
			if !assert.NoError(err) {
				return
			}
			assert.Len(tt.chart.Files, 1)
			assert.Equal(tt.want.servicePath, testUnit.Service.Path())
			assert.Equal(tt.want.serviceFile, string(testUnit.Service.Program()))
			assert.Equal(testUnit.Service, tt.chart.Files[0])
			assert.Equal(tt.want.values, values)
		})
	}
}

func Test_addTargetGroupBinding(t *testing.T) {
	type TestUnit struct {
		targetGroupBindingPath string
		targetGroupBindingFile string
		values                 []Value
	}
	tests := []struct {
		name  string
		chart KlothoHelmChart
		want  TestUnit
	}{
		{
			name: "happy path test",
			chart: KlothoHelmChart{
				Name: "test",
				ExecutionUnits: []*HelmExecUnit{
					{
						Name:      "unit",
						Namespace: "default",
					},
				},
			},
			want: TestUnit{
				targetGroupBindingPath: "test/templates/unit-targetgroupbinding.yaml",
				targetGroupBindingFile: `apiVersion: elbv2.k8s.aws/v1beta1
kind: TargetGroupBinding
metadata:
  creationTimestamp: null
  labels:
    execUnit: unit
  name: unit
spec:
  serviceRef:
    name: unit
    port: 80
  targetGroupARN: '{{ .Values.unitTargetGroupArn }}'
status: {}
`,
				values: []Value{
					{
						ExecUnitName: "unit",
						Kind:         "TargetGroupBinding",
						Type:         "target_group",
						Key:          "unitTargetGroupArn",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			testUnit := tt.chart.ExecutionUnits[0]
			values, err := tt.chart.addTargetGroupBinding(testUnit)
			if !assert.NoError(err) {
				return
			}
			assert.Len(tt.chart.Files, 1)
			assert.Equal(tt.want.targetGroupBindingPath, testUnit.TargetGroupBinding.Path())
			assert.Equal(tt.want.targetGroupBindingFile, string(testUnit.TargetGroupBinding.Program()))
			assert.Equal(testUnit.TargetGroupBinding, tt.chart.Files[0])
			assert.Equal(tt.want.values, values)
		})
	}
}

func Test_addServiceExport(t *testing.T) {
	type TestUnit struct {
		targetGroupBindingPath string
		targetGroupBindingFile string
	}
	tests := []struct {
		name  string
		chart KlothoHelmChart
		want  TestUnit
	}{
		{
			name: "happy path test",
			chart: KlothoHelmChart{
				Name: "test",
				ExecutionUnits: []*HelmExecUnit{
					{
						Name:      "unit",
						Namespace: "default",
					},
				},
			},
			want: TestUnit{
				targetGroupBindingPath: "test/templates/unit-serviceexport.yaml",
				targetGroupBindingFile: `kind: ServiceExport
apiVersion: multicluster.x-k8s.io/v1alpha1
metadata:
  namespace: default
  name: unit
`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			testUnit := tt.chart.ExecutionUnits[0]
			err := tt.chart.addServiceExport(testUnit)
			if !assert.NoError(err) {
				return
			}
			assert.Len(tt.chart.Files, 1)
			assert.Equal(tt.want.targetGroupBindingPath, testUnit.ServiceExport.Path())
			assert.Equal(tt.want.targetGroupBindingFile, string(testUnit.ServiceExport.Program()))
			assert.Equal(testUnit.ServiceExport, tt.chart.Files[0])
		})
	}
}
