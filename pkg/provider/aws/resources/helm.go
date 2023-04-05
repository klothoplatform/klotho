package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/infra/kubernetes"
)

type (
	AwsHelmChart struct {
		Name               string
		Chart              string
		Directory          string
		Files              []core.File
		Values             AwsHelmChartValues `render:"template"`
		ConstructRefs      []core.AnnotationKey
		FetchOpts          HelmFetchOpts `render:"template"`
		KubernetesProvider *AwsKubernetesProvider
		Namespace          string
		Version            string
		EnvVarKeys         map[string]string
	}

	AwsHelmChartValues struct {
		Values []AwsHelmChartValue
	}

	// Values specifies the values that exist in the generated helm chart, which are necessary to provide during installation to run on the provider
	AwsHelmChartValue struct {
		Key   string // Key is the key to be used in helms values.yaml file or cli
		Value core.IaCValue
	}

	HelmFetchOpts struct {
		Repo string
	}
)

// Provider returns name of the provider the resource is correlated to
func (chart *AwsHelmChart) Provider() string { return "aws" }

// KlothoConstructRef returns a slice containing the ids of any Klotho constructs is correlated to
func (chart *AwsHelmChart) KlothoConstructRef() []core.AnnotationKey { return chart.ConstructRefs }

func (chart *AwsHelmChart) Id() string {
	return fmt.Sprintf("aws_klotho_helm_chart-%s", chart.Name)
}

func NewAwsHelmChart(khchart *kubernetes.HelmChart) *AwsHelmChart {
	chart := &AwsHelmChart{
		Name:          khchart.Name,
		Chart:         khchart.Chart,
		Directory:     khchart.Directory,
		Files:         khchart.Files,
		ConstructRefs: khchart.ConstructRefs,
		EnvVarKeys:    map[string]string{},
	}

	for _, val := range khchart.Values {
		if val.EnvironmentVariable != nil {
			chart.EnvVarKeys[val.EnvironmentVariable.GetName()] = val.Key
		}
	}

	return chart
}

// TODO look into a better way to represent the k8s provider since it's more of a pulumi construct
type AwsKubernetesProvider struct {
	ConstructRefs []core.AnnotationKey
	KubeConfig    string
	Name          string
}

func (e AwsKubernetesProvider) Provider() string {
	return "aws"
}

func (e AwsKubernetesProvider) KlothoConstructRef() []core.AnnotationKey {
	return e.ConstructRefs
}

func (e AwsKubernetesProvider) Id() string {
	return fmt.Sprintf("%s:%s:%s", e.Provider(), "eks_provider", e.Name)
}
