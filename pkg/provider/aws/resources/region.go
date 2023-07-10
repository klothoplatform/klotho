package resources

import (
	"github.com/klothoplatform/klotho/pkg/core"
)

type (
	Region struct {
		Name          string
		ConstructRefs core.BaseConstructSet `yaml:"-"`
	}

	AvailabilityZones struct {
		Name          string
		ConstructRefs core.BaseConstructSet `yaml:"-"`
	}

	AccountId struct {
		Name          string
		ConstructRefs core.BaseConstructSet `yaml:"-"`
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
		ConstructRefs: core.BaseConstructSetOf(),
	}
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (region *Region) BaseConstructRefs() core.BaseConstructSet {
	return region.ConstructRefs
}

// Id returns the id of the cloud resource
func (region *Region) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     REGION_TYPE,
		Name:     REGION_NAME,
	}
}
func (region *Region) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}

func NewAvailabilityZones() *AvailabilityZones {
	return &AvailabilityZones{
		Name:          AVAILABILITY_ZONES_NAME,
		ConstructRefs: core.BaseConstructSetOf(),
	}
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (azs *AvailabilityZones) BaseConstructRefs() core.BaseConstructSet {
	return azs.ConstructRefs
}

// Id returns the id of the cloud resource
func (azs *AvailabilityZones) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     AVAILABILITY_ZONES_TYPE,
		Name:     AVAILABILITY_ZONES_NAME,
	}
}

func (azs *AvailabilityZones) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}

func NewAccountId() *AccountId {
	return &AccountId{
		Name:          ACCOUNT_ID_NAME,
		ConstructRefs: core.BaseConstructSetOf(),
	}
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (id *AccountId) BaseConstructRefs() core.BaseConstructSet {
	return id.ConstructRefs
}

// Id returns the id of the cloud resource
func (id *AccountId) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     ACCOUNT_ID_TYPE,
		Name:     ACCOUNT_ID_NAME,
	}
}

func (id *AccountId) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}
