package aws

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/construct"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"gopkg.in/yaml.v3"
)

type AWS struct {
	AppName string
}

func (a *AWS) Name() string { return provider.AWS }

func (a *AWS) ListResources() []construct.Resource {
	return resources.ListAll()
}

// CreateResourceFromId creates a resource from an id, but does not mutate the graph in any manner
// The graph is passed in to be able to understand what namespaces reference in resource ids
func (a *AWS) CreateConstructFromId(id construct.ResourceId, dag *construct.ConstructGraph) (construct.BaseConstruct, error) {
	typeToResource := make(map[string]construct.Resource)
	for _, res := range resources.ListAll() {
		typeToResource[res.Id().Type] = res
	}
	// Subnets are special because they have a type that is not the same as their resource type since it uses a characteristic of the subnet
	typeToResource["subnet_private"] = &resources.Subnet{}
	typeToResource["subnet_public"] = &resources.Subnet{}
	res, ok := typeToResource[id.Type]
	if !ok {
		return nil, fmt.Errorf("unable to find resource of type %s", id.Type)
	}
	newResource := reflect.New(reflect.TypeOf(res).Elem()).Interface()
	resource, ok := newResource.(construct.Resource)
	if !ok {
		return nil, fmt.Errorf("item %s of type %T is not of type construct.Resource", id, newResource)
	}
	reflect.ValueOf(resource).Elem().FieldByName("Name").SetString(id.Name)
	if subnet, ok := resource.(*resources.Subnet); ok {
		switch id.Type {
		case "subnet_public":
			subnet.Type = resources.PublicSubnet
		case "subnet_private":
			subnet.Type = resources.PrivateSubnet
		}
	}

	if id.Namespace != "" {
		method := reflect.ValueOf(resource).MethodByName("Load")
		if method.IsValid() {
			var callArgs []reflect.Value
			callArgs = append(callArgs, reflect.ValueOf(id.Namespace))
			callArgs = append(callArgs, reflect.ValueOf(dag))
			eval := method.Call(callArgs)
			if !eval[0].IsNil() {
				err, ok := eval[0].Interface().(error)
				if !ok {
					return nil, fmt.Errorf("return type should be an error")
				}
				if err != nil {
					return nil, err
				}
			}
		}
	}
	return resource, nil
}

//go:embed resources/templates/*
var awsTempaltes embed.FS

func (a *AWS) GetOperationalTemplates() map[construct.ResourceId]*knowledgebase.ResourceTemplate {
	templates := map[construct.ResourceId]*knowledgebase.ResourceTemplate{}
	if err := fs.WalkDir(awsTempaltes, ".", func(path string, d fs.DirEntry, nerr error) error {
		if d.IsDir() {
			return nil
		}
		content, err := awsTempaltes.ReadFile(fmt.Sprintf("resources/templates/%s", d.Name()))
		if err != nil {
			panic(err)
		}
		resTemplate := &knowledgebase.ResourceTemplate{}
		err = yaml.Unmarshal(content, resTemplate)
		if err != nil {
			panic(err)
		}
		id := construct.ResourceId{Provider: provider.AWS, Type: resTemplate.Type}
		if templates[id] != nil {
			panic(fmt.Errorf("duplicate template for type %s", resTemplate.Type))
		}
		templates[id] = resTemplate
		return nil
	}); err != nil {
		return templates
	}
	return templates
}

//go:embed edges/*
var awsEdgeTempaltes embed.FS

func (a *AWS) GetEdgeTemplates() map[string]*knowledgebase.EdgeTemplate {
	templates := map[string]*knowledgebase.EdgeTemplate{}
	err := fs.WalkDir(awsEdgeTempaltes, ".", func(path string, d fs.DirEntry, nerr error) error {
		if d.IsDir() {
			return nil
		}
		content, err := awsEdgeTempaltes.ReadFile(fmt.Sprintf("edges/%s", d.Name()))
		if err != nil {
			return errors.Join(nerr, err)
		}
		resTemplate := &knowledgebase.EdgeTemplate{}
		err = yaml.Unmarshal(content, resTemplate)
		if err != nil {
			return errors.Join(nerr, fmt.Errorf("unable to unmarshal edge template %s: %w", d.Name(), err))
		}
		templateKey := resTemplate.Key()
		if templates[templateKey] != nil {
			return errors.Join(nerr, fmt.Errorf("duplicate template for type %s", templateKey))
		}
		templates[templateKey] = resTemplate
		return nil
	})
	if err != nil {
		panic(err)
	}
	return templates
}
