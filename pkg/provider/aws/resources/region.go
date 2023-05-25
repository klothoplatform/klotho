package resources

import (
	"github.com/klothoplatform/klotho/pkg/core"
)

type (
	Region struct {
		Name          string
		ConstructsRef core.AnnotationKeySet
	}

	AvailabilityZones struct {
		Name          string
		ConstructsRef core.AnnotationKeySet
	}

	AccountId struct {
		Name          string
		ConstructsRef core.AnnotationKeySet
	}
)

const (
	REGION_NAME             = "region"
	AVAILABILITY_ZONES_NAME = "AvailabilityZones"
	ACCOUNT_ID_NAME         = "AccountId"
	REGION_TYPE             = "region"
	AVAILABILITY_ZONES_TYPE = "availability_zones"
	ACCOUNT_ID_TYPE         = "account_id"
	ARN_IAC_VALUE           = "arn"
)

func NewRegion() *Region {
	return &Region{
		Name:          REGION_NAME,
		ConstructsRef: core.AnnotationKeySetOf(),
	}
}

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (region *Region) KlothoConstructRef() core.AnnotationKeySet {
	return region.ConstructsRef
}

// Id returns the id of the cloud resource
func (region *Region) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     REGION_TYPE,
		Name:     REGION_NAME,
	}
}

func NewAvailabilityZones() *AvailabilityZones {
	return &AvailabilityZones{
		Name:          AVAILABILITY_ZONES_NAME,
		ConstructsRef: core.AnnotationKeySetOf(),
	}
}

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (azs *AvailabilityZones) KlothoConstructRef() core.AnnotationKeySet {
	return azs.ConstructsRef
}

// Id returns the id of the cloud resource
func (azs *AvailabilityZones) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     AVAILABILITY_ZONES_TYPE,
		Name:     AVAILABILITY_ZONES_NAME,
	}
}

func NewAccountId() *AccountId {
	return &AccountId{
		Name:          ACCOUNT_ID_NAME,
		ConstructsRef: core.AnnotationKeySetOf(),
	}
}

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (id *AccountId) KlothoConstructRef() core.AnnotationKeySet {
	return id.ConstructsRef
}

// Id returns the id of the cloud resource
func (id *AccountId) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     ACCOUNT_ID_TYPE,
		Name:     ACCOUNT_ID_NAME,
	}
}
