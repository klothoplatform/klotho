package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

type (
	Secret struct {
		Name          string
		SecretName    string
		ConstructsRef []core.AnnotationKey
	}

	SecretVersion struct {
		SecretName    string
		Secret        *Secret
		Path          string
		ConstructsRef []core.AnnotationKey
		Name          string
		Type          string
	}
)

const SECRET_TYPE = "secret"
const SECRET_VERSION_TYPE = "secret_version"

type SecretCreateParams struct {
	AppName string
	Refs    []core.AnnotationKey
	Name    string
}

// Create takes in an all necessary parameters to generate the Secret name and ensure that the Secret is correlated to the constructs which required its creation.
func (secret *Secret) Create(dag *core.ResourceGraph, params SecretCreateParams) error {
	secret.Name = aws.SecretSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	secret.ConstructsRef = params.Refs
	existingSecret := dag.GetResource(secret.Id())
	if existingSecret != nil {
		return fmt.Errorf("Secret with name %s already exists", secret.Name)
	}
	dag.AddResource(secret)
	return nil
}

type SecretVersionCreateParams struct {
	AppName string
	Refs    []core.AnnotationKey
	Name    string
}

// Create takes in an all necessary parameters to generate the SecretVersion name and ensure that the SecretVersion is correlated to the constructs which required its creation.
//
// This method will also create dependent resources which are necessary for functionality. Those resources are:
//   - Secret
func (sv *SecretVersion) Create(dag *core.ResourceGraph, params SecretVersionCreateParams) error {
	sv.Name = aws.SecretSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	sv.ConstructsRef = params.Refs
	existingSecret := dag.GetResource(sv.Id())
	if existingSecret != nil {
		return fmt.Errorf("SecretVersion with name %s already exists", sv.Name)
	}
	err := dag.CreateDependencies(sv, map[string]any{
		"Secret": params,
	})
	if err != nil {
		return err
	}
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
	sv.SecretName = sv.Secret.Name
	return nil
}

func NewSecret(annot core.AnnotationKey, secretName string, appName string) *Secret {
	plainName := appName
	if secretName != "" {
		plainName += "-" + secretName
	}
	return &Secret{
		Name:          plainName,
		SecretName:    aws.SecretSanitizer.Apply(plainName),
		ConstructsRef: []core.AnnotationKey{annot},
	}
}

func NewSecretVersion(secret *Secret, filePath string) *SecretVersion {
	return &SecretVersion{
		SecretName:    secret.SecretName,
		Secret:        secret,
		Path:          filePath,
		ConstructsRef: secret.ConstructsRef,
		Name:          secret.Name,
	}
}

func (s *Secret) KlothoConstructRef() []core.AnnotationKey {
	return s.ConstructsRef
}

func (s *Secret) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     SECRET_TYPE,
		Name:     s.Name,
	}
}

func (sv *SecretVersion) KlothoConstructRef() []core.AnnotationKey {
	return sv.ConstructsRef
}

func (sv *SecretVersion) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     SECRET_VERSION_TYPE,
		Name:     sv.Name,
	}
}
