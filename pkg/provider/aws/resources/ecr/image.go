package ecr

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

const ECR_IMAGE_TYPE = "ecr_image"

type (
	EcrImage struct {
		Name          string
		ConstructsRef []core.AnnotationKey
		Context       string
		Dockerfile    string
		ExtraOptions  []string
	}
)

func NewEcrImage(unit *core.ExecutionUnit, appName string) *EcrImage {
	return &EcrImage{
		Name:          fmt.Sprintf("%s-%s", appName, unit.ID),
		ConstructsRef: []core.AnnotationKey{unit.Provenance()},
		Context:       fmt.Sprintf("./%s", unit.ID),
		Dockerfile:    fmt.Sprintf("./%s/%s", unit.ID, unit.DockerfilePath),
		ExtraOptions:  []string{"--platform", "linux/amd64", "--quiet"},
	}
}

// Provider returns name of the provider the resource is correlated to
func (image *EcrImage) Provider() string {
	return resources.AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (image *EcrImage) KlothoConstructRef() []core.AnnotationKey {
	return image.ConstructsRef
}

// ID returns the id of the cloud resource
func (image *EcrImage) Id() string {
	return fmt.Sprintf("%s_%s", ECR_IMAGE_TYPE, image.Name)
}
