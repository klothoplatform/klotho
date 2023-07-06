package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/engine/classification"
)

const (
	ROUTE_53_HOSTED_ZONE_TYPE  = "route53_hosted_zone"
	ROUTE_53_RECORD_TYPE       = "route53_record"
	ROUTE_53_HEALTH_CHECK_TYPE = "route53_health_check"
)

type (
	Route53HostedZone struct {
		Name          string
		ConstructsRef core.BaseConstructSet `yaml:"-"`
		Vpcs          []*Vpc
		ForceDestroy  bool
	}

	Route53Record struct {
		Name          string
		DomainName    string
		ConstructsRef core.BaseConstructSet `yaml:"-"`
		Zone          *Route53HostedZone
		Type          string
		Records       []*AwsResourceValue
		HealthCheck   *Route53HealthCheck
		TTL           int
	}

	Route53HealthCheck struct {
		Name             string
		ConstructsRef    core.BaseConstructSet `yaml:"-"`
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
	zone.ConstructsRef = params.Refs

	existingZone, found := core.GetResource[*Route53HostedZone](dag, zone.Id())
	if found {
		existingZone.ConstructsRef.AddAll(params.Refs)
		return nil
	}
	dag.AddResource(zone)
	return nil
}

func (zone *Route53HostedZone) MakeOperational(dag *core.ResourceGraph, appName string, classifier classification.Classifier) error {
	vpcs := core.GetDownstreamResourcesOfType[*Vpc](dag, zone)
	for _, vpc := range vpcs {
		if !collectionutil.Contains(zone.Vpcs, vpc) {
			zone.Vpcs = append(zone.Vpcs, vpc)
		}
	}
	dag.AddDependenciesReflect(zone)
	return nil
}

type Route53HostedZoneConfigureParams struct {
}

func (zone *Route53HostedZone) Configure(params Route53HostedZoneConfigureParams) error {
	zone.ForceDestroy = true
	return nil
}

type Route53RecordCreateParams struct {
	Refs       core.BaseConstructSet
	DomainName string
	Zone       *Route53HostedZone
}

func (record *Route53Record) Create(dag *core.ResourceGraph, params Route53RecordCreateParams) error {
	record.Name = fmt.Sprintf("%s-%s", params.Zone.Name, params.DomainName)
	record.ConstructsRef = params.Refs
	record.DomainName = params.DomainName

	existingRecord, found := core.GetResource[*Route53Record](dag, record.Id())
	if found {
		existingRecord.ConstructsRef.AddAll(params.Refs)
		return nil
	}
	record.Zone = params.Zone
	dag.AddDependenciesReflect(record)
	return nil
}

func (record *Route53Record) MakeOperational(dag *core.ResourceGraph, appName string, classifier classification.Classifier) error {
	if record.Zone == nil {
		zones := core.GetDownstreamResourcesOfType[*Route53HostedZone](dag, record)
		if len(zones) != 1 {
			return fmt.Errorf("Route53Record %s has %d zone dependencies", record.Name, len(zones))
		}
		record.Zone = zones[0]
	}
	dag.AddDependenciesReflect(record)
	return nil
}

type Route53RecordConfigureParams struct {
	Type        string
	HealthCheck *Route53HealthCheck
	TTL         int
}

func (record *Route53Record) Configure(params Route53RecordConfigureParams) error {
	if params.Type != "" {
		record.Type = params.Type
	}
	if params.HealthCheck != nil {
		record.HealthCheck = params.HealthCheck
	}
	if params.TTL != 0 {
		record.TTL = params.TTL
	}
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
	healthCheck.ConstructsRef = params.Refs

	existingHealthCheck, found := core.GetResource[*Route53HealthCheck](dag, healthCheck.Id())
	if found {
		existingHealthCheck.ConstructsRef.AddAll(params.Refs)
		return nil
	}
	dag.AddDependenciesReflect(healthCheck)
	return nil
}

type Route53HealthCheckConfigureParams struct {
	Type             string
	Disabled         bool
	FailureThreshold int
	Port             int
	RequestInterval  int
	ResourcePath     string
}

func (healthCheck *Route53HealthCheck) Configure(params Route53HealthCheckConfigureParams) error {
	if params.Type != "" {
		healthCheck.Type = params.Type
	}
	healthCheck.Disabled = params.Disabled
	if params.FailureThreshold != 0 {
		healthCheck.FailureThreshold = params.FailureThreshold
	}
	if params.Port != 0 {
		healthCheck.Port = params.Port
	}
	if params.RequestInterval != 0 {
		healthCheck.RequestInterval = params.RequestInterval
	}
	if params.ResourcePath != "" {
		healthCheck.ResourcePath = params.ResourcePath
	}
	return nil
}

// BaseConstructsRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (zone *Route53HostedZone) BaseConstructsRef() core.BaseConstructSet {
	return zone.ConstructsRef
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

func (record *Route53Record) BaseConstructsRef() core.BaseConstructSet {
	return record.ConstructsRef
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

func (hc *Route53HealthCheck) BaseConstructsRef() core.BaseConstructSet {
	return hc.ConstructsRef
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
