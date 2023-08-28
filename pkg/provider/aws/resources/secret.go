package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

type (
	Secret struct {
		Name          string
		ConstructRefs construct.BaseConstructSet `yaml:"-"`
	}

	SecretVersion struct {
		Secret        *Secret
		DetectedPath  string
		Path          string
		ConstructRefs construct.BaseConstructSet `yaml:"-"`
		Name          string
		Type          string
	}
)

const SECRET_TYPE = "secret"
const SECRET_VERSION_TYPE = "secret_version"

type SecretCreateParams struct {
	AppName string
	Refs    construct.BaseConstructSet
	Name    string
}

// Create takes in an all necessary parameters to generate the Secret name and ensure that the Secret is correlated to the constructs which required its creation.
func (s *Secret) Create(dag *construct.ResourceGraph, params SecretCreateParams) error {
	s.Name = aws.SecretSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	s.ConstructRefs = params.Refs.Clone()
	if existingSecret, ok := construct.GetResource[*Secret](dag, s.Id()); ok {
		existingSecret.ConstructRefs.AddAll(params.Refs)
		return nil
	}
	dag.AddResource(s)
	return nil
}

type SecretVersionCreateParams struct {
	AppName      string
	Refs         construct.BaseConstructSet
	Name         string
	DetectedPath string
}

// Create takes in an all necessary parameters to generate the SecretVersion name and ensure that the SecretVersion is correlated to the constructs which required its creation.
//
// This method will also create dependent resources which are necessary for functionality. Those resources are:
//   - Secret
func (sv *SecretVersion) Create(dag *construct.ResourceGraph, params SecretVersionCreateParams) error {
	sv.Name = aws.SecretSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	sv.ConstructRefs = params.Refs.Clone()
	sv.DetectedPath = params.DetectedPath
	existingSecret := dag.GetResource(sv.Id())
	if existingSecret != nil {
		existingSecret.(*SecretVersion).ConstructRefs.AddAll(params.Refs)
		return nil
	}
	dag.AddResource(sv)
	return nil
}

func (s *Secret) BaseConstructRefs() construct.BaseConstructSet {
	return s.ConstructRefs
}

func (s *Secret) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     SECRET_TYPE,
		Name:     s.Name,
	}
}

func (s *Secret) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream:     true,
		RequiresNoDownstream:   true,
		RequiresExplicitDelete: true,
	}
}

func (sv *SecretVersion) BaseConstructRefs() construct.BaseConstructSet {
	return sv.ConstructRefs
}

func (sv *SecretVersion) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     SECRET_VERSION_TYPE,
		Name:     sv.Name,
	}
}
func (sv *SecretVersion) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream: true,
	}
}
