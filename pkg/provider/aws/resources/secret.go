package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/engine/classification"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

type (
	Secret struct {
		Name          string
		ConstructsRef core.BaseConstructSet `yaml:"-"`
	}

	SecretVersion struct {
		Secret        *Secret
		DetectedPath  string
		Path          string
		ConstructsRef core.BaseConstructSet `yaml:"-"`
		Name          string
		Type          string
	}
)

const SECRET_TYPE = "secret"
const SECRET_VERSION_TYPE = "secret_version"

type SecretCreateParams struct {
	AppName string
	Refs    core.BaseConstructSet
	Name    string
}

// Create takes in an all necessary parameters to generate the Secret name and ensure that the Secret is correlated to the constructs which required its creation.
func (s *Secret) Create(dag *core.ResourceGraph, params SecretCreateParams) error {
	s.Name = aws.SecretSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	s.ConstructsRef = params.Refs.Clone()
	if existingSecret, ok := core.GetResource[*Secret](dag, s.Id()); ok {
		return fmt.Errorf("secret with name %s already exists", existingSecret.Name)
	}
	dag.AddResource(s)
	return nil
}

type SecretVersionCreateParams struct {
	AppName      string
	Refs         core.BaseConstructSet
	Name         string
	DetectedPath string
}

// Create takes in an all necessary parameters to generate the SecretVersion name and ensure that the SecretVersion is correlated to the constructs which required its creation.
//
// This method will also create dependent resources which are necessary for functionality. Those resources are:
//   - Secret
func (sv *SecretVersion) Create(dag *core.ResourceGraph, params SecretVersionCreateParams) error {
	sv.Name = aws.SecretSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	sv.ConstructsRef = params.Refs.Clone()
	sv.DetectedPath = params.DetectedPath
	existingSecret := dag.GetResource(sv.Id())
	if existingSecret != nil {
		return fmt.Errorf("SecretVersion with name %s already exists", sv.Name)
	}
	dag.AddResource(sv)
	return nil
}

func (sv *SecretVersion) MakeOperational(dag *core.ResourceGraph, appName string, classifier classification.Classifier) error {
	if sv.Secret == nil {
		versions := core.GetDownstreamResourcesOfType[*Secret](dag, sv)
		if len(versions) > 1 {
			return fmt.Errorf("SecretVersion %s has multiple Secret dependencies", sv.Name)
		} else if len(versions) == 0 {
			err := dag.CreateDependencies(sv, map[string]any{
				"Secret": SecretCreateParams{
					AppName: appName,
					Refs:    core.BaseConstructSetOf(sv),
					Name:    sv.Name,
				},
			})
			if err != nil {
				return err
			}
		} else {
			sv.Secret = versions[0]
		}
	}
	dag.AddDependenciesReflect(sv)
	return nil

}

type SecretVersionConfigureParams struct {
	Type string
	Path string
}

// Configure sets the intristic characteristics of a vpc based on parameters passed in
func (sv *SecretVersion) Configure(params SecretVersionConfigureParams) error {
	sv.Type = params.Type
	sv.Path = params.Path
	return nil
}

func (s *Secret) BaseConstructsRef() core.BaseConstructSet {
	return s.ConstructsRef
}

func (s *Secret) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     SECRET_TYPE,
		Name:     s.Name,
	}
}

func (s *Secret) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream:     true,
		RequiresNoDownstream:   true,
		RequiresExplicitDelete: true,
	}
}

func (sv *SecretVersion) BaseConstructsRef() core.BaseConstructSet {
	return sv.ConstructsRef
}

func (sv *SecretVersion) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     SECRET_VERSION_TYPE,
		Name:     sv.Name,
	}
}
func (sv *SecretVersion) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}
