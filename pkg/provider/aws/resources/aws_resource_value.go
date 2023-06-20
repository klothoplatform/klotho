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

	valueRaw struct {
		Type     string
		Resource core.Resource
		Property string
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
	type intermediate struct {
		Type     string
		Resource map[interface{}]interface{}
		Property string
	}
	in := intermediate{}
	err := value.Decode(&in)
	if err != nil {
		return err
	}
	val.PropertyVal = in.Property
	if in.Type == "" {
		return nil
	}

	typeToResource := make(map[string]core.Resource)
	for _, res := range ListAll() {
		typeToResource[res.Id().Type] = res
	}
	// Subnets are special because they have a type that is not the same as their resource type since it uses a characteristic of the subnet
	typeToResource["subnet_private"] = &Subnet{}
	typeToResource["subnet_public"] = &Subnet{}

	res := typeToResource[in.Type]
	md, err := yaml.Marshal(in.Resource)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(md, res)
	if err != nil {
		return err
	}
	val.ResourceVal = res

	return nil
}

func (val *AwsResourceValue) MarshalYAML() (interface{}, error) {
	if val.ResourceVal == nil {
		return valueRaw{
			Property: val.PropertyVal,
		}, nil
	}
	return valueRaw{
		Type:     val.ResourceVal.Id().Type,
		Resource: val.ResourceVal,
		Property: val.PropertyVal,
	}, nil
}
