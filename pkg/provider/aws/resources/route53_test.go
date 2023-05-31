package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_Route53HostedZoneCreate(t *testing.T) {
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}}
	initialRefs := core.AnnotationKeySetOf(core.AnnotationKey{ID: "first"})
	cases := []coretesting.CreateCase[Route53HostedZoneCreateParams, *Route53HostedZone]{
		{
			Name: "nil private zone",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:route53_hosted_zone:my-app-zone",
					"aws:vpc:my_app",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:route53_hosted_zone:my-app-zone", Destination: "aws:vpc:my_app"},
				},
			},
			Params: Route53HostedZoneCreateParams{Type: "private"},
			Check: func(assert *assert.Assertions, zone *Route53HostedZone) {
				assert.Equal(zone.Name, "my-app-zone")
				assert.NotNil(zone.Vpc)
				assert.Equal(zone.ConstructsRef, core.AnnotationKeySetOf(eu.AnnotationKey))
			},
		},
		{
			Name: "nil public zone",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:route53_hosted_zone:my-app-zone",
				},
				Deps: []coretesting.StringDep{},
			},
			Params: Route53HostedZoneCreateParams{Type: "public"},
			Check: func(assert *assert.Assertions, zone *Route53HostedZone) {
				assert.Equal(zone.Name, "my-app-zone")
				assert.Nil(zone.Vpc)
				assert.Equal(zone.ConstructsRef, core.AnnotationKeySetOf(eu.AnnotationKey))
			},
		},
		{
			Name:     "existing load balancer",
			Existing: &Route53HostedZone{Name: "my-app-zone", ConstructsRef: initialRefs},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:route53_hosted_zone:my-app-zone",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, zone *Route53HostedZone) {
				assert.Equal(zone.Name, "my-app-zone")
				initialRefs.Add(eu.AnnotationKey)
				assert.Equal(zone.ConstructsRef, initialRefs)
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = Route53HostedZoneCreateParams{
				AppName: "my-app",
				Refs:    core.AnnotationKeySetOf(eu.AnnotationKey),
				Type:    tt.Params.Type,
				Name:    "zone",
			}
			tt.Run(t)
		})
	}
}

func Test_Route53RecordCreate(t *testing.T) {
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}}
	initialRefs := core.AnnotationKeySetOf(core.AnnotationKey{ID: "first"})
	cases := []coretesting.CreateCase[Route53RecordCreateParams, *Route53Record]{
		{
			Name: "nil private zone",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:route53_hosted_zone:my-app-zone",
					"aws:route53_record:my-app-zone-record",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:route53_record:my-app-zone-record", Destination: "aws:route53_hosted_zone:my-app-zone"},
				},
			},
			Check: func(assert *assert.Assertions, record *Route53Record) {
				assert.Equal(record.Name, "my-app-zone-record")
				assert.NotNil(record.Zone)
				assert.Equal(record.ConstructsRef, core.AnnotationKeySetOf(eu.AnnotationKey))
			},
		},
		{
			Name:     "existing load balancer",
			Existing: &Route53Record{Name: "my-app-zone-record", ConstructsRef: initialRefs},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:route53_record:my-app-zone-record",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, record *Route53Record) {
				assert.Equal(record.Name, "my-app-zone-record")
				initialRefs.Add(eu.AnnotationKey)
				assert.Equal(record.ConstructsRef, initialRefs)
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = Route53RecordCreateParams{
				Refs:       core.AnnotationKeySetOf(eu.AnnotationKey),
				Zone:       &Route53HostedZone{Name: "my-app-zone", ConstructsRef: initialRefs},
				DomainName: "record",
			}
			tt.Run(t)
		})
	}
}

func Test_Route53HealthCheckCreate(t *testing.T) {
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}}
	initialRefs := core.AnnotationKeySetOf(core.AnnotationKey{ID: "first"})
	cases := []coretesting.CreateCase[Route53HealthCheckCreateParams, *Route53HealthCheck]{
		{
			Name: "nil check ip",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:route53_health_check:my-app-10.0.0.0",
				},
				Deps: []coretesting.StringDep{},
			},
			Params: Route53HealthCheckCreateParams{IpAddress: "10.0.0.0"},
			Check: func(assert *assert.Assertions, healthCheck *Route53HealthCheck) {
				assert.Equal(healthCheck.Name, "my-app-10.0.0.0")
				assert.Equal(healthCheck.ConstructsRef, core.AnnotationKeySetOf(eu.AnnotationKey))
			},
		},
		{
			Name: "nil check fqdn",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:route53_health_check:my-app-example.com",
				},
				Deps: []coretesting.StringDep{},
			},
			Params: Route53HealthCheckCreateParams{Fqdn: "example.com"},
			Check: func(assert *assert.Assertions, healthCheck *Route53HealthCheck) {
				assert.Equal(healthCheck.Name, "my-app-example.com")
				assert.Equal(healthCheck.ConstructsRef, core.AnnotationKeySetOf(eu.AnnotationKey))
			},
		},
		{
			Name:     "existing health check",
			Existing: &Route53HealthCheck{Name: "my-app-check", ConstructsRef: initialRefs},
			Params:   Route53HealthCheckCreateParams{Fqdn: "check"},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:route53_health_check:my-app-check",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, healthCheck *Route53HealthCheck) {
				assert.Equal(healthCheck.Name, "my-app-check")
				initialRefs.Add(eu.AnnotationKey)
				assert.Equal(healthCheck.ConstructsRef, initialRefs)
			},
		},
		{
			Name:     "ip and fqdn error",
			Params:   Route53HealthCheckCreateParams{IpAddress: "10.0.0.0", Fqdn: "example.com"},
			Existing: &Route53HealthCheck{Name: "my-app-check", ConstructsRef: initialRefs},
			WantErr:  true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = Route53HealthCheckCreateParams{
				Refs:      core.AnnotationKeySetOf(eu.AnnotationKey),
				AppName:   "my-app",
				IpAddress: tt.Params.IpAddress,
				Fqdn:      tt.Params.Fqdn,
			}
			tt.Run(t)
		})
	}
}
