package resources

import (
	"github.com/klothoplatform/klotho/pkg/construct"
)

type (
	Region struct {
		Name          string
		ConstructRefs construct.BaseConstructSet `yaml:"-"`
	}

	AvailabilityZones struct {
		Name          string
		ConstructRefs construct.BaseConstructSet `yaml:"-"`
	}

	AccountId struct {
		Name          string
		ConstructRefs construct.BaseConstructSet `yaml:"-"`
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

var availabilityZones = []string{"0", "1"}

func NewRegion() *Region {
	return &Region{
		Name:          REGION_NAME,
		ConstructRefs: construct.BaseConstructSetOf(),
	}
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (region *Region) BaseConstructRefs() construct.BaseConstructSet {
	return region.ConstructRefs
}

// Id returns the id of the cloud resource
func (region *Region) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     REGION_TYPE,
		Name:     REGION_NAME,
	}
}
func (region *Region) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream: true,
	}
}

func NewAvailabilityZones() *AvailabilityZones {
	return &AvailabilityZones{
		Name:          AVAILABILITY_ZONES_NAME,
		ConstructRefs: construct.BaseConstructSetOf(),
	}
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (azs *AvailabilityZones) BaseConstructRefs() construct.BaseConstructSet {
	return azs.ConstructRefs
}

// Id returns the id of the cloud resource
func (azs *AvailabilityZones) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     AVAILABILITY_ZONES_TYPE,
		Name:     AVAILABILITY_ZONES_NAME,
	}
}

func (azs *AvailabilityZones) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream: true,
	}
}

func NewAccountId() *AccountId {
	return &AccountId{
		Name:          ACCOUNT_ID_NAME,
		ConstructRefs: construct.BaseConstructSetOf(),
	}
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (id *AccountId) BaseConstructRefs() construct.BaseConstructSet {
	return id.ConstructRefs
}

// Id returns the id of the cloud resource
func (id *AccountId) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     ACCOUNT_ID_TYPE,
		Name:     ACCOUNT_ID_NAME,
	}
}

func (id *AccountId) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream: true,
	}
}
