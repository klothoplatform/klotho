package resources

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"gopkg.in/yaml.v3"
)

type (
	AwsResourceValue struct {
		ResourceVal core.Resource
		PropertyVal string
	}
)

func (r *AwsResourceValue) Resource() core.Resource {
	return r.ResourceVal
}

func (r *AwsResourceValue) Property() string {
	return r.PropertyVal
}

func (r *AwsResourceValue) SetResource(res core.Resource) {
	r.ResourceVal = res
}

func (val *AwsResourceValue) UnmarshalYAML(value *yaml.Node) error {
	return nil
}

func (val *AwsResourceValue) MarshalYAML() (interface{}, error) {
	return nil, nil
}
