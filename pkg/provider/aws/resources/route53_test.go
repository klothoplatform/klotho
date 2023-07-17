package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_Route53HostedZoneCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test"}
	eu2 := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu2)
	cases := []coretesting.CreateCase[Route53HostedZoneCreateParams, *Route53HostedZone]{
		{
			Name: "nil zone",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:route53_hosted_zone:my-app-zone",
				},
				Deps: []coretesting.StringDep{},
			},
			Params: Route53HostedZoneCreateParams{Type: "private"},
			Check: func(assert *assert.Assertions, zone *Route53HostedZone) {
				assert.Equal(zone.Name, "my-app-zone")
				assert.Equal(zone.ConstructRefs, core.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "existing zone",
			Existing: &Route53HostedZone{Name: "my-app-zone", ConstructRefs: initialRefs},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:route53_hosted_zone:my-app-zone",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, zone *Route53HostedZone) {
				assert.Equal(zone.Name, "my-app-zone")
				initialRefs.Add(eu)
				assert.Equal(zone.ConstructRefs, initialRefs)
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = Route53HostedZoneCreateParams{
				AppName: "my-app",
				Refs:    core.BaseConstructSetOf(eu),
				Type:    tt.Params.Type,
				Name:    "zone",
			}
			tt.Run(t)
		})
	}
}

func Test_Route53RecordCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test"}
	eu2 := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu2)
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
				assert.Equal(record.ConstructRefs, core.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "existing load balancer",
			Existing: &Route53Record{Name: "my-app-zone-record", ConstructRefs: initialRefs},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:route53_record:my-app-zone-record",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, record *Route53Record) {
				assert.Equal(record.Name, "my-app-zone-record")
				initialRefs.Add(eu)
				assert.Equal(record.ConstructRefs, initialRefs)
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = Route53RecordCreateParams{
				Refs:       core.BaseConstructSetOf(eu),
				Zone:       &Route53HostedZone{Name: "my-app-zone", ConstructRefs: initialRefs},
				DomainName: "record",
			}
			tt.Run(t)
		})
	}
}

func Test_Route53HealthCheckCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test"}
	eu2 := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu2)
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
				assert.Equal(healthCheck.ConstructRefs, core.BaseConstructSetOf(eu))
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
				assert.Equal(healthCheck.ConstructRefs, core.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "existing health check",
			Existing: &Route53HealthCheck{Name: "my-app-check", ConstructRefs: initialRefs},
			Params:   Route53HealthCheckCreateParams{Fqdn: "check"},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:route53_health_check:my-app-check",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, healthCheck *Route53HealthCheck) {
				assert.Equal(healthCheck.Name, "my-app-check")
				initialRefs.Add(eu)
				assert.Equal(healthCheck.ConstructRefs, initialRefs)
			},
		},
		{
			Name:     "ip and fqdn error",
			Params:   Route53HealthCheckCreateParams{IpAddress: "10.0.0.0", Fqdn: "example.com"},
			Existing: &Route53HealthCheck{Name: "my-app-check", ConstructRefs: initialRefs},
			WantErr:  true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = Route53HealthCheckCreateParams{
				Refs:      core.BaseConstructSetOf(eu),
				AppName:   "my-app",
				IpAddress: tt.Params.IpAddress,
				Fqdn:      tt.Params.Fqdn,
			}
			tt.Run(t)
		})
	}
}
