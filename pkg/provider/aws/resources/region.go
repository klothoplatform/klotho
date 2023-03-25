package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
)

const REGION_TYPE = "region"

type (
	Region struct {
		Name          string
		ConstructsRef []core.AnnotationKey
	}
)

func NewRegion() *Region {
	return &Region{
		Name:          "region",
		ConstructsRef: []core.AnnotationKey{},
	}
}

// Provider returns name of the provider the resource is correlated to
func (region *Region) Provider() string {
	return AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (region *Region) KlothoConstructRef() []core.AnnotationKey {
	return region.ConstructsRef
}

// ID returns the id of the cloud resource
func (region *Region) Id() string {
	return fmt.Sprintf("%s:%s:%s", region.Provider(), REGION_TYPE, region.Name)
}
