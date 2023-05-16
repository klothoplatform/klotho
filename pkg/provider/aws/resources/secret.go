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
		SecretName            string
		Secret                *Secret
		Path                  string
		ConstructsRef         []core.AnnotationKey
		secretNameUnSanitized string
		Type                  string
	}
)

const SECRET_TYPE = "secret"
const SECRET_VERSION_TYPE = "secret_version"

type SecretCreateParams struct {
	AppName    string
	Refs       []core.AnnotationKey
	SecretName string
}

func (secret *Secret) Create(dag *core.ResourceGraph, params SecretCreateParams) error {
	secret.Name = aws.SecretSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.SecretName))
	secret.ConstructsRef = params.Refs
	dag.AddResource(secret)
	return nil
}

type SecretVersionCreateParams struct {
	AppName    string
	Refs       []core.AnnotationKey
	SecretName string
	Path       string
	Type       string
}

func (sv *SecretVersion) Create(dag *core.ResourceGraph, params SecretVersionCreateParams) error {
	sv.SecretName = aws.SecretSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.SecretName))
	sv.Path = params.Path
	sv.ConstructsRef = params.Refs
	sv.Type = params.Type
	dag.CreateDependencies(sv, map[string]any{
		"Secret": params,
	})
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
		SecretName:            secret.SecretName,
		Secret:                secret,
		Path:                  filePath,
		ConstructsRef:         secret.ConstructsRef,
		secretNameUnSanitized: secret.Name,
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
		Name:     sv.secretNameUnSanitized,
	}
}
