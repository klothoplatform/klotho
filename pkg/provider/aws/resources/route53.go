package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
)

const (
	ROUTE_53_HOSTED_ZONE_TYPE  = "route53_hosted_zone"
	ROUTE_53_RECORD_TYPE       = "route53_record"
	ROUTE_53_HEALTH_CHECK_TYPE = "route53_health_check"
)

type (
	Route53HostedZone struct {
		Name          string
		ConstructRefs core.BaseConstructSet `yaml:"-"`
		Vpcs          []*Vpc
		ForceDestroy  bool
	}

	Route53Record struct {
		Name          string
		DomainName    string
		ConstructRefs core.BaseConstructSet `yaml:"-"`
		Zone          *Route53HostedZone
		Type          string
		Records       []core.IaCValue
		HealthCheck   *Route53HealthCheck
		TTL           int
	}

	Route53HealthCheck struct {
		Name             string
		ConstructRefs    core.BaseConstructSet `yaml:"-"`
		Type             string
		Disabled         bool
		FailureThreshold int
		Fqdn             string
		IpAddress        string
		Port             int
		RequestInterval  int
		ResourcePath     string
	}
)

type Route53HostedZoneCreateParams struct {
	AppName string
	Refs    core.BaseConstructSet
	Name    string
	Type    string
}

func (zone *Route53HostedZone) Create(dag *core.ResourceGraph, params Route53HostedZoneCreateParams) error {
	zone.Name = fmt.Sprintf("%s-%s", params.AppName, params.Name)
	zone.ConstructRefs = params.Refs

	existingZone, found := core.GetResource[*Route53HostedZone](dag, zone.Id())
	if found {
		existingZone.ConstructRefs.AddAll(params.Refs)
		return nil
	}
	dag.AddResource(zone)
	return nil
}

type Route53RecordCreateParams struct {
	Refs       core.BaseConstructSet
	DomainName string
	Zone       *Route53HostedZone
}

func (record *Route53Record) Create(dag *core.ResourceGraph, params Route53RecordCreateParams) error {
	record.Name = fmt.Sprintf("%s-%s", params.Zone.Name, params.DomainName)
	record.ConstructRefs = params.Refs
	record.DomainName = params.DomainName

	existingRecord, found := core.GetResource[*Route53Record](dag, record.Id())
	if found {
		existingRecord.ConstructRefs.AddAll(params.Refs)
		return nil
	}
	record.Zone = params.Zone
	dag.AddDependenciesReflect(record)
	return nil
}

type Route53HealthCheckCreateParams struct {
	Refs      core.BaseConstructSet
	AppName   string
	Fqdn      string
	IpAddress string
}

func (healthCheck *Route53HealthCheck) Create(dag *core.ResourceGraph, params Route53HealthCheckCreateParams) error {
	if params.Fqdn != "" && params.IpAddress != "" {
		return fmt.Errorf("cannot set fully qualified domain name and ip address on route53 health check")
	}
	name := fmt.Sprintf("%s-%s", params.AppName, params.IpAddress)
	healthCheck.IpAddress = params.IpAddress
	if params.IpAddress == "" {
		name = fmt.Sprintf("%s-%s", params.AppName, params.Fqdn)
		healthCheck.Fqdn = params.Fqdn
	}
	healthCheck.Name = name
	healthCheck.ConstructRefs = params.Refs

	existingHealthCheck, found := core.GetResource[*Route53HealthCheck](dag, healthCheck.Id())
	if found {
		existingHealthCheck.ConstructRefs.AddAll(params.Refs)
		return nil
	}
	dag.AddDependenciesReflect(healthCheck)
	return nil
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (zone *Route53HostedZone) BaseConstructRefs() core.BaseConstructSet {
	return zone.ConstructRefs
}

// Id returns the id of the cloud resource
func (zone *Route53HostedZone) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     ROUTE_53_HOSTED_ZONE_TYPE,
		Name:     zone.Name,
	}
}

func (zone *Route53HostedZone) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}

func (record *Route53Record) BaseConstructRefs() core.BaseConstructSet {
	return record.ConstructRefs
}

// Id returns the id of the cloud resource
func (record *Route53Record) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     ROUTE_53_RECORD_TYPE,
		Name:     record.Name,
	}
}

func (record *Route53Record) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}

func (hc *Route53HealthCheck) BaseConstructRefs() core.BaseConstructSet {
	return hc.ConstructRefs
}

// Id returns the id of the cloud resource
func (hc *Route53HealthCheck) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     ROUTE_53_HEALTH_CHECK_TYPE,
		Name:     hc.Name,
	}
}

func (record *Route53HealthCheck) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream:   true,
		RequiresNoDownstream: true,
	}
}
