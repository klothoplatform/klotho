package helm

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/chart"
)

type chartOptions struct {
	*chart.Chart
}

type chartOption func(*chartOptions)

func buildChart(opts ...chartOption) *chart.Chart {
	c := &chartOptions{
		Chart: &chart.Chart{
			Metadata: &chart.Metadata{
				APIVersion: "v1",
				Name:       "hello",
				Version:    "0.1.0",
			},
			// This adds a basic template.
			Templates: []*chart.File{
				{Name: "templates/sa.yaml", Data: []byte(`apiVersion: v1
kind: ServiceAccount
metadata:
	name: {{ .Values.ServiceAccount.Name }}
	namespace: {{ .Release.Namespace }}`)},
			},
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	// This adds a basic crd.
	c.Files = append(c.Files, &chart.File{
		Name: "crds/crd.yaml",
		Data: []byte(`apiVersion: "apiextensions.k8s.io/v1beta1"
kind: "CustomResourceDefinition"
metadata:
	name: "sampleCrd
spec:
	group: "example.martin-helmich.de"
	version: "v1alpha1"
	scope: "Namespaced"
	names:
	plural: "projects"
	singular: "project"
	kind: "Project"
	validation:
	openAPIV3Schema:
		required: ["spec"]
		properties:
		spec:
			required: ["replicas"]
			properties:
			replicas:
				type: "integer"
				minimum: 1`),
	})

	return c.Chart
}

func buildValues() map[string]interface{} {
	return map[string]interface{}{
		"ServiceAccount": map[string]interface{}{
			"Name": "TestSa",
		},
	}
}

func Test_GetRenderedTemplates(t *testing.T) {
	tests := []struct {
		name      string
		fileUnits map[string]string
		want      string
	}{
		{
			name: "Load Basic Chart",
			fileUnits: map[string]string{
				"Chart.yaml":           "",
				"templates/pod.yaml":   "",
				"crds/custom_pod.yaml": "",
			},
			want: `apiVersion: v1
kind: ServiceAccount
metadata:
	name: TestSa
	namespace: TEST_NAMESPACE`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			ch := buildChart()
			values := buildValues()

			h, err := NewHelmHelper()
			if !assert.NoError(err) {
				return
			}
			files, err := h.GetRenderedTemplates(ch, values, "TEST_NAMESPACE")
			if !assert.NoError(err) {
				return
			}

			assert.Len(files, 2)
			for _, f := range files {

				assert.Contains([]string{"hello/templates/sa.yaml", "hello/crds/crd.yaml"}, f.Path())

				if f.Path() == "hello/templates/sa.yaml" {
					ast, ok := f.(*types.SourceFile)
					if !assert.True(ok) {
						return
					}
					assert.Equal(tt.want, string(ast.Program()))
				}

			}
		})
	}
}
