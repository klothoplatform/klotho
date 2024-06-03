package constructs

import (
	"gopkg.in/yaml.v3"
	"testing"
)

func TestReadConstructTemplateFromYamlString(t *testing.T) {
	// Define the YAML string
	yamlString := `
id: klotho.aws.Container
version: 0.0.1
description: A container that runs a task
inputs:
  foo:
    type: number
    description: bar
    default: 10
    secret: false
resources:
  baz:
    type: aws:s3_bucket
    name: my-bucket
  bop:
    type: aws:ecs_service
    name: my-service
    properties:
      Cpu: ${input:foo}
edges:
 - from: baz
   to: bop
   data: { action: read}
`

	// Create a new ConstructTemplate instance
	var c ConstructTemplate

	// Unmarshal the YAML string into the ConstructTemplate instance
	if err := yaml.Unmarshal([]byte(yamlString), &c); err != nil {
		t.Fatalf("Failed to unmarshal YAML string: %v", err)
	}

	//context := ConstructContext{
	//	ConstructTemplate: ConstructTemplate{},
	//	Inputs: map[string]any{
	//		"foo":      10,
	//		"keyInput": "my-key",
	//		"Array":    []string{"a", "b", "c"},
	//		"Map": map[string]any{
	//			"key1": "value1",
	//		},
	//	},
	//}

	//val, err := context.InterpolateResource([]string{"x-y-${inputs:foo}-${inputs:foo}-cool", "${inputs:foo}", "cool"})
	//t.Logf("Interpolated value: %v", val)
	//val, err = context.InterpolateResource(map[string]string{
	//	"${inputs:keyInput}": "x-y-${inputs:foo}-${inputs:foo}-cool", "key2": "${inputs:foo}", "key3": "cool"})
	//t.Logf("Interpolated value: %v", val)
	//val, err = context.InterpolateResource("${inputs:Map.key1}")
	//t.Logf("Interpolated value: %v", val)
	//val, err = context.InterpolateResource("${inputs:Array[0]}-${inputs:Array[1]}-${inputs:Array[2]}")
	//t.Logf("Interpolated value: %v", val)

	//interpolate the ConstructTemplate instance

	// print the ConstructTemplate instance as yaml
	out, err := yaml.Marshal(&c)
	if err != nil {
		t.Fatalf("Failed to marshal ConstructTemplate instance: %v", err)
	}
	t.Logf("Constructed ConstructTemplate instance:\n%s", string(out))

}

func TestReadConstructTemplateFromYamlFile(t *testing.T) {
	id, _ := ParseConstructTemplateId("klotho.aws.Container")
	template := loadConstructTemplate(id)
	t.Logf("Constructed ConstructTemplate instance:\n%v", template)
}

func TestParse(t *testing.T) {
	// Define the YAML string
	ctx := NewContext(
		map[string]any{
			"Image": "nginx:latest",
		},
		ConstructId{
			TemplateId: ConstructTemplateId{
				Package: "klotho.aws",
				Name:    "Container",
			},
			InstanceId: "my-instance",
		},
	)
	resolvedConstruct := ctx.evaluateConstruct()
	//t.Logf("Resolved Construct: %v", resolvedConstruct)
	// log the resolve construct as yaml
	out, err := yaml.Marshal(&resolvedConstruct)
	t.Logf("Resolved Construct:\n%s", string(out))
	if err != nil {
		t.Fatalf("Failed to resolve Construct: %v", err)
	}

	cs, err := (&ConstructMarshaller{
		Construct: resolvedConstruct,
		Context:   ctx,
	}).Marshal()
	csyaml, err := yaml.Marshal(&cs)
	t.Logf("Constraints:\n%s", string(csyaml))
}
