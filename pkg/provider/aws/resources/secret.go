package resources

import (
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
