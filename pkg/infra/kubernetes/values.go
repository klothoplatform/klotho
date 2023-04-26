package kubernetes

import "github.com/klothoplatform/klotho/pkg/core"

// Values specifies the values that exist in the generated helm chart, which are necessary to provide during installation to run on the provider
type HelmChartValue struct {
	ExecUnitName        string                   // ExecUnitName signifies the exec unit that this value is for
	Kind                string                   // Kind is the kind of the kubernetes object this value is applied to
	Type                string                   // Type is the type of value expected
	Key                 string                   // Key is the key to be used in helms values.yaml file or cli
	EnvironmentVariable core.EnvironmentVariable //EnvironmentVariable provides information around what environment variable is needed for substitution
}

type ProviderValueTypes string

const (
	TargetGroupTransformation              ProviderValueTypes = "target_group"
	ImageTransformation                    ProviderValueTypes = "image"
	ImageHashTransformation                ProviderValueTypes = "image_hash"
	EnvironmentVariableTransformation      ProviderValueTypes = "env_var"
	ServiceAccountAnnotationTransformation ProviderValueTypes = "service_account_annotation"
	InstanceTypeKey                        ProviderValueTypes = "instance_type_key"
	InstanceTypeValue                      ProviderValueTypes = "instance_type_value"
)
