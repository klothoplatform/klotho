package kubernetes

import (
	"bytes"
	"strings"
	"testing"

	"github.com/klothoplatform/klotho/pkg/testutil"

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
		hpaPath            string
	}
	tests := []struct {
		name      string
		fileUnits map[string]string
		units     []string
		want      []TestUnit
		wantErr   bool
	}{
		{
			name:  "Basic Pod",
			units: []string{"unit1"},
			fileUnits: map[string]string{
				"pod.yaml": testutil.UnIndent(`
                    apiVersion: v1
                    kind: Pod
                    spec:
                      containers:
                      - name: web
                        image: nginx`)},
			want: []TestUnit{
				{
					name:    "unit1",
					podPath: "pod.yaml",
				},
			},
		},
		{
			name:  "Basic Deployment",
			units: []string{"unit1"},
			fileUnits: map[string]string{
				"deployment.yaml": testutil.UnIndent(`
                    apiVersion: apps/v1
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
                            image: nginx:1.14.2`)},
			want: []TestUnit{
				{
					name:           "unit1",
					deploymentPath: "deployment.yaml",
				},
			},
		},
		{
			name:  "Basic ServiceAccount",
			units: []string{"unit1"},
			fileUnits: map[string]string{
				"ServiceAccount.yaml": testutil.UnIndent(`
                    apiVersion: v1
                    kind: ServiceAccount
                    metadata:
                      name: release-name-nginx-ingress
                      namespace: default`)},
			want: []TestUnit{
				{
					name:               "unit1",
					serviceAccountPath: "ServiceAccount.yaml",
				},
			},
		},
		{
			name:  "Basic Service",
			units: []string{"unit1"},
			fileUnits: map[string]string{
				"Service.yaml": testutil.UnIndent(`
                    apiVersion: v1
                    kind: Service
                    spec:
                      ports:
                      - port: 80
                        protocol: TCP
                        targetPort: 3000
                      selector:
                        execUnit: name`)},
			want: []TestUnit{
				{
					name:        "unit1",
					servicePath: "Service.yaml",
				},
			},
		},
		{
			name:  "Basic HPA",
			units: []string{"unit1"},
			fileUnits: map[string]string{
				"HorizontalPodAutoscaler.yaml": testutil.UnIndent(`
                    apiVersion: autoscaling/v2
                    kind: HorizontalPodAutoscaler
                    metadata:
                      name: example-hpa
                    spec:
                      scaleTargetRef:
                        apiVersion: apps/v1
                        kind: Deployment
                        name: example-deployment
                      minReplicas: 2
                      maxReplicas: 10
                      metrics:
                      - type: Resource
                        resource:
                          name: cpu
                          targetAverageUtilization: 50`)},
			want: []TestUnit{
				{
					name:    "unit1",
					hpaPath: "HorizontalPodAutoscaler.yaml",
				},
			},
		},
		{
			name:  "Multi unit Pod",
			units: []string{"unit1", "unit2"},
			fileUnits: map[string]string{
				"pod.yaml": testutil.UnIndent(`
                    apiVersion: v1
                    kind: Pod
                    metadata:
                      name: unit1
                    spec:
                      containers:
                      - name: web
                        image: nginx`),
				"pod2.yaml": testutil.UnIndent(`
                    apiVersion: v1
                    kind: Pod
                    metadata:
                      name: notunit2
                    spec:
                      containers:
                      - name: web
                        image: nginx`)},
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
			name:  "multi unit Deployment",
			units: []string{"unit1", "unit2"},
			fileUnits: map[string]string{
				"deployment.yaml": testutil.UnIndent(`
                    apiVersion: apps/v1
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
                            image: nginx:1.14.2`),
				"deployment2.yaml": testutil.UnIndent(`
                    apiVersion: apps/v1
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
                            image: nginx:1.14.2`),
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
			name:  "multi  unit HPA",
			units: []string{"unit1", "unit2"},
			fileUnits: map[string]string{
				"HorizontalPodAutoscaler.yaml": testutil.UnIndent(`
                    apiVersion: autoscaling/v2
                    kind: HorizontalPodAutoscaler
                    metadata:
                      name: unit1
                    spec:
                      scaleTargetRef:
                        apiVersion: apps/v1
                        kind: Deployment
                        name: example-deployment
                      minReplicas: 2
                      maxReplicas: 10
                      metrics:
                      - type: Resource
                        resource:
                          name: cpu
                          targetAverageUtilization: 50`)},
			want: []TestUnit{
				{
					name:    "unit1",
					hpaPath: "HorizontalPodAutoscaler.yaml",
				},
				{
					name: "unit2",
				},
			},
		},
		{
			name:  "multi unit ServiceAccount",
			units: []string{"unit1", "unit2"},
			fileUnits: map[string]string{
				"ServiceAccount.yaml": testutil.UnIndent(`
                    apiVersion: v1
                    kind: ServiceAccount
                    metadata:
                      name: unit1
                      namespace: default`),
				"ServiceAccount2.yaml": testutil.UnIndent(`
                    apiVersion: v1
                    kind: ServiceAccount
                    metadata:
                      name: notunit2
                      namespace: default`),
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
			name:  "multi unit Service",
			units: []string{"unit1", "unit2"},
			fileUnits: map[string]string{
				"Service.yaml": testutil.UnIndent(`
                    apiVersion: v1
                    kind: Service
                    metadata:
                      name: unit1
                    spec:
                      ports:
                      - port: 80
                        protocol: TCP
                        targetPort: 3000
                      selector:
                        execUnit: name`),
				"Service2.yaml": testutil.UnIndent(`
                    apiVersion: v1
                    kind: Service
                    metadata:
                      name: notunit2
                    spec:
                      ports:
                      - port: 80
                        protocol: TCP
                        targetPort: 3000
                      selector:
                        execUnit: name`),
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
			name:  "single unit pod and deployment error",
			units: []string{"unit1"},
			fileUnits: map[string]string{
				"pod.yaml": testutil.UnIndent(`
                    apiVersion: v1
                    kind: Pod
                    spec:
                      containers:
                      - name: web
                        image: nginx`),
				"deployment.yaml": testutil.UnIndent(`
                    apiVersion: apps/v1
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
                            image: nginx:1.14.2`),
			},
			wantErr: true,
		},
		{
			name:  "multi unit pod and deployment error",
			units: []string{"unit1", "unit2"},
			fileUnits: map[string]string{
				"pod.yaml": testutil.UnIndent(`
                    apiVersion: v1
                    kind: Pod
                    metadata:
                      name: unit1
                    spec:
                      containers:
                      - name: web
                        image: nginx`),
				"deployment.yaml": testutil.UnIndent(`
                    apiVersion: apps/v1
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
                              image: nginx:1.14.2`),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			chart := &HelmChart{
				Name:      "test",
				Namespace: "default",
			}

			for _, name := range tt.units {
				chart.ExecutionUnits = append(chart.ExecutionUnits, &HelmExecUnit{Name: name})
			}

			for path, file := range tt.fileUnits {
				f, err := yamlLang.NewFile(path, strings.NewReader(file))
				if assert.Nil(err) {
					chart.Files = append(chart.Files, f)
				}
			}

			err := chart.AssignFilesToUnits()
			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			for _, hu := range tt.want {
				for _, cu := range chart.ExecutionUnits {
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
						if hu.hpaPath != "" {
							assert.Equal(hu.hpaPath, cu.HorizontalPodAutoscaler.Path())
						} else {
							assert.Nil(cu.HorizontalPodAutoscaler)
						}
					}
				}
			}
		})
	}
}

func Test_handleExecutionUnit(t *testing.T) {
	testUnitName := "unit"
	tests := []struct {
		name          string
		hasDockerfile bool
		cfg           config.ExecutionUnit
		want          []HelmChartValue
		wantErr       bool
	}{
		{
			name:          "no transforms",
			hasDockerfile: false,
			cfg:           config.ExecutionUnit{},
			want:          []HelmChartValue{},
		},
		{
			name:          "only dockerfile",
			hasDockerfile: true,
			cfg:           config.ExecutionUnit{},
			want: []HelmChartValue{
				{
					ExecUnitName: testUnitName,
					Kind:         "Deployment",
					Type:         "image",
					Key:          "unitImage",
				},
				{
					ExecUnitName: testUnitName,
					Kind:         "ServiceAccount",
					Type:         "service_account_annotation",
					Key:          "unitRoleArn",
				},
			},
		},
		{
			name:          "network placement",
			hasDockerfile: true,
			cfg:           config.ExecutionUnit{NetworkPlacement: "private"},
			want: []HelmChartValue{
				{
					ExecUnitName: testUnitName,
					Kind:         "Deployment",
					Type:         "image",
					Key:          "unitImage",
				},
				{
					ExecUnitName: testUnitName,
					Kind:         "ServiceAccount",
					Type:         "service_account_annotation",
					Key:          "unitRoleArn",
				},
			},
		},
		{
			name:          "node group",
			hasDockerfile: true,
			cfg:           config.ExecutionUnit{NetworkPlacement: "private", InfraParams: config.ConvertToInfraParams(config.KubernetesTypeParams{NodeType: "node", InstanceType: "test.node"})},
			want: []HelmChartValue{
				{
					ExecUnitName: testUnitName,
					Kind:         "Deployment",
					Type:         "image",
					Key:          "unitImage",
				},
				{
					ExecUnitName: testUnitName,
					Kind:         "Deployment",
					Type:         "instance_type_key",
					Key:          "unitInstanceTypeKey",
				},
				{
					ExecUnitName: testUnitName,
					Kind:         "Deployment",
					Type:         "instance_type_value",
					Key:          "unitInstanceTypeValue",
				},
				{
					ExecUnitName: testUnitName,
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
			eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: testUnitName, Capability: annotation.ExecutionUnitCapability}}
			if tt.hasDockerfile {
				dockerF, err := dockerfile.NewFile("Dockerfile", bytes.NewBuffer([]byte{}))
				if !assert.NoError(err) {
					return
				}
				eu.Add(dockerF)
			}
			constructGraph := core.NewConstructGraph()

			testUnit := &HelmExecUnit{Name: eu.ID}
			chart := &HelmChart{
				Name:           "test",
				Namespace:      "default",
				ExecutionUnits: []*HelmExecUnit{testUnit},
			}
			transformations, err := chart.handleExecutionUnit(testUnit, eu, tt.cfg, constructGraph)
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
		values []HelmChartValue
		files  []string
	}
	tests := []struct {
		name    string
		unit    *HelmExecUnit
		deps    []graph.Edge[core.Construct]
		want    testResult
		wantErr bool
	}{
		{
			name: "gateway dep",
			unit: &HelmExecUnit{Name: "unit", Namespace: "default"},
			deps: []graph.Edge[core.Construct]{
				{
					Source:      &core.Gateway{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExposeCapability}},
					Destination: &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{Capability: annotation.ExecutionUnitCapability, ID: "unit"}},
				},
			},
			want: testResult{
				files: []string{"test/templates/unit-targetgroupbinding.yaml"},
				values: []HelmChartValue{
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
			unit: &HelmExecUnit{Name: "unit", Namespace: "default"},
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
			unit: &HelmExecUnit{Name: "unit", Namespace: "default"},
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
				values: []HelmChartValue{
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
			unit: &HelmExecUnit{Name: "unit", Namespace: "default"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			constructGraph := core.NewConstructGraph()
			for _, dep := range tt.deps {
				constructGraph.AddConstruct(dep.Source)
				constructGraph.AddConstruct(dep.Destination)
				constructGraph.AddDependency(dep.Source.Id(), dep.Destination.Id())
			}
			chart := HelmChart{
				Name:           "test",
				ExecutionUnits: []*HelmExecUnit{tt.unit},
			}
			values, err := chart.handleUpstreamUnitDependencies(tt.unit, constructGraph, config.ExecutionUnit{})
			if tt.wantErr {
				assert.Error(err)
				return
			} else if !assert.NoError(err) {
				return

			}
			assert.Equal(tt.want.values, values)
			for _, f := range tt.want.files {
				if strings.Contains(f, "targetgroupbinding") {
					assert.Equal(f, tt.unit.TargetGroupBinding.Path())
				}
				if strings.Contains(f, "serviceexport") {
					assert.Equal(f, tt.unit.ServiceExport.Path())
				}
			}
		})
	}
}

func Test_addDeployment(t *testing.T) {
	type TestUnit struct {
		deploymentPath string
		deploymentFile string
		values         []HelmChartValue
	}
	tests := []struct {
		name    string
		unit    *HelmExecUnit
		cfg     config.ExecutionUnit
		want    TestUnit
		wantErr bool
	}{
		{
			name: "happy path test",
			unit: &HelmExecUnit{Name: "unit", Namespace: "default"},
			want: TestUnit{
				deploymentPath: "test/templates/unit-deployment.yaml",
				deploymentFile: `apiVersion: apps/v1
kind: Deployment
metadata:
  creationTimestamp: null
  labels:
    execUnit: unit
    klotho-fargate-enabled: "false"
  name: unit
  namespace: default
spec:
  replicas: 2
  selector:
    matchLabels:
      execUnit: unit
      klotho-fargate-enabled: "false"
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
				values: []HelmChartValue{
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
			chart := &HelmChart{
				Name:           "test",
				Namespace:      "default",
				ExecutionUnits: []*HelmExecUnit{tt.unit},
			}
			values, err := chart.addDeployment(tt.unit, tt.cfg)
			if !assert.NoError(err) {
				return
			}
			assert.Len(chart.Files, 1)
			assert.Equal(tt.want.deploymentPath, tt.unit.Deployment.Path())
			assert.Equal(tt.want.deploymentFile, string(tt.unit.Deployment.Program()))
			assert.Equal(tt.unit.Deployment, chart.Files[0])
			assert.Equal(tt.want.values, values)
		})
	}
}

func Test_addHorizontalPodAutoscaler(t *testing.T) {
	type result struct {
		hpaPath string
		hpaFile string
		values  []HelmChartValue
	}
	tests := []struct {
		name  string
		chart HelmChart
		want  result
	}{
		{
			name: "happy path test",
			chart: HelmChart{
				Name: "test",
				ExecutionUnits: []*HelmExecUnit{
					{
						Name:      "unit",
						Namespace: "default",
					},
				},
			},
			want: result{
				hpaPath: "test/templates/unit-horizontal-pod-autoscaler.yaml",
				hpaFile: testutil.UnIndent(`
                    apiVersion: autoscaling/v2
                    kind: HorizontalPodAutoscaler
                    metadata:
                      creationTimestamp: null
                      labels:
                        execUnit: unit
                      name: unit
                    spec:
                      maxReplicas: 4
                      metrics:
                      - resource:
                          name: cpu
                          target:
                            averageUtilization: 70
                            type: Utilization
                        type: Resource
                      - resource:
                          name: memory
                          target:
                            averageUtilization: 70
                            type: Utilization
                        type: Resource
                      minReplicas: 2
                      scaleTargetRef:
                        apiVersion: apps/v1
                        kind: Deployment
                        name: unit
                    status:
                      currentMetrics: null
                      desiredReplicas: 0`),
				values: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			testUnit := tt.chart.ExecutionUnits[0]
			values, err := tt.chart.addHorizontalPodAutoscaler(testUnit, config.ExecutionUnit{})
			if !assert.NoError(err) {
				return
			}
			assert.Len(tt.chart.Files, 1)
			assert.Equal(tt.want.hpaPath, testUnit.HorizontalPodAutoscaler.Path())
			assert.Equal(tt.want.hpaFile, string(testUnit.HorizontalPodAutoscaler.Program()))
			assert.Equal(testUnit.HorizontalPodAutoscaler, tt.chart.Files[0])
			assert.Equal(tt.want.values, values)
		})
	}
}

func Test_addServiceAccount(t *testing.T) {
	type TestUnit struct {
		serviceAccountPath string
		serviceAccountFile string
		values             []HelmChartValue
	}
	tests := []struct {
		name string
		unit *HelmExecUnit
		want TestUnit
	}{
		{
			name: "happy path test",
			unit: &HelmExecUnit{Name: "unit", Namespace: "default"},
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
				values: []HelmChartValue{
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
			chart := &HelmChart{
				Name:           "test",
				Namespace:      "default",
				ExecutionUnits: []*HelmExecUnit{tt.unit},
			}

			values, err := chart.addServiceAccount(tt.unit)
			if !assert.NoError(err) {
				return
			}
			assert.Len(chart.Files, 1)
			assert.Equal(tt.want.serviceAccountPath, tt.unit.ServiceAccount.Path())
			assert.Equal(tt.want.serviceAccountFile, string(tt.unit.ServiceAccount.Program()))
			assert.Equal(tt.unit.ServiceAccount, chart.Files[0])
			assert.Equal(tt.want.values, values)
		})
	}
}

func Test_addService(t *testing.T) {
	type TestUnit struct {
		servicePath string
		serviceFile string
		values      []HelmChartValue
	}
	tests := []struct {
		name string
		unit *HelmExecUnit
		want TestUnit
	}{
		{
			name: "happy path test",
			unit: &HelmExecUnit{Name: "unit", Namespace: "default"},
			want: TestUnit{
				servicePath: "test/templates/unit-service.yaml",
				serviceFile: `apiVersion: v1
kind: Service
metadata:
  creationTimestamp: null
  labels:
    execUnit: unit
    klotho-fargate-enabled: "false"
  name: unit
  namespace: default
spec:
  ports:
  - port: 80
    protocol: TCP
    targetPort: 3000
  selector:
    execUnit: unit
    klotho-fargate-enabled: "false"
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
			chart := &HelmChart{
				Name:           "test",
				Namespace:      "default",
				ExecutionUnits: []*HelmExecUnit{tt.unit},
			}

			values, err := chart.addService(tt.unit, config.ExecutionUnit{})
			if !assert.NoError(err) {
				return
			}
			assert.Len(chart.Files, 1)
			assert.Equal(tt.want.servicePath, tt.unit.Service.Path())
			assert.Equal(tt.want.serviceFile, string(tt.unit.Service.Program()))
			assert.Equal(tt.unit.Service, chart.Files[0])
			assert.Equal(tt.want.values, values)
		})
	}
}

func Test_addTargetGroupBinding(t *testing.T) {
	testUnitName := "unit"
	type TestUnit struct {
		targetGroupBindingPath string
		targetGroupBindingFile string
		values                 []HelmChartValue
	}
	tests := []struct {
		name string
		want TestUnit
	}{
		{
			name: "happy path test",
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
				values: []HelmChartValue{
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
			testUnit := &HelmExecUnit{Name: testUnitName}
			chart := &HelmChart{
				Name:           "test",
				Namespace:      "default",
				ExecutionUnits: []*HelmExecUnit{testUnit},
			}

			values, err := chart.addTargetGroupBinding(testUnit)
			if !assert.NoError(err) {
				return
			}
			assert.Len(chart.Files, 1)
			assert.Equal(tt.want.targetGroupBindingPath, testUnit.TargetGroupBinding.Path())
			assert.Equal(tt.want.targetGroupBindingFile, string(testUnit.TargetGroupBinding.Program()))
			assert.Equal(testUnit.TargetGroupBinding, chart.Files[0])
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
		name string
		unit *HelmExecUnit
		want TestUnit
	}{
		{
			name: "happy path test",
			unit: &HelmExecUnit{Name: "unit", Namespace: "default"},
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
			chart := &HelmChart{
				Name:           "test",
				Namespace:      "default",
				ExecutionUnits: []*HelmExecUnit{tt.unit},
			}

			err := chart.addServiceExport(tt.unit)
			if !assert.NoError(err) {
				return
			}
			assert.Len(chart.Files, 1)
			assert.Equal(tt.want.targetGroupBindingPath, tt.unit.ServiceExport.Path())
			assert.Equal(tt.want.targetGroupBindingFile, string(tt.unit.ServiceExport.Program()))
			assert.Equal(tt.unit.ServiceExport, chart.Files[0])
		})
	}
}
