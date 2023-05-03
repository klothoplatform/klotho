package resources

import (
	"github.com/klothoplatform/klotho/pkg/core"
)

const REGION_TYPE = "region"
const AVAILABILITY_ZONES_TYPE = "availability_zones"
const ACCOUNT_ID_TYPE = "account_id"
const ARN_IAC_VALUE = "arn"

type (
	Region struct {
		Name          string
		ConstructsRef []core.AnnotationKey
	}

	AvailabilityZones struct {
		Name          string
		ConstructsRef []core.AnnotationKey
	}

	AccountId struct {
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

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (region *Region) KlothoConstructRef() []core.AnnotationKey {
	return region.ConstructsRef
}

// Id returns the id of the cloud resource
func (region *Region) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     REGION_TYPE,
		Name:     region.Name,
	}
}

func NewAvailabilityZones() *AvailabilityZones {
	return &AvailabilityZones{
		Name:          "AvailabilityZones",
		ConstructsRef: []core.AnnotationKey{},
	}
}

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (azs *AvailabilityZones) KlothoConstructRef() []core.AnnotationKey {
	return azs.ConstructsRef
}

// Id returns the id of the cloud resource
func (azs *AvailabilityZones) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     AVAILABILITY_ZONES_TYPE,
		Name:     azs.Name,
	}
}

func NewAccountId() *AccountId {
	return &AccountId{
		Name:          "AccountId",
		ConstructsRef: []core.AnnotationKey{},
	}
}

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (id *AccountId) KlothoConstructRef() []core.AnnotationKey {
	return id.ConstructsRef
}

// Id returns the id of the cloud resource
func (id *AccountId) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     ACCOUNT_ID_TYPE,
		Name:     id.Name,
	}
}
