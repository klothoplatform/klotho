package aws

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/klothoplatform/klotho/pkg/graph"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/stretchr/testify/assert"
)

func Test_createCDNs(t *testing.T) {
	appName := "test"
	cdn1 := "cdn"
	cdn2 := "otherCdn"
	su := &core.StaticUnit{AnnotationKey: core.AnnotationKey{ID: "su"}}
	gw := &core.Gateway{AnnotationKey: core.AnnotationKey{ID: "gw"}}
	distro := resources.NewCloudfrontDistribution(appName, cdn1)
	distro2 := resources.NewCloudfrontDistribution(appName, cdn2)

	cases := []struct {
		name                   string
		constructs             []core.Construct
		constructIdToResources map[string][]core.Resource
		cfg                    config.Application
		want                   coretesting.ResourcesExpectation
		wantRefs               map[core.ResourceId][]core.AnnotationKey
		wantErr                bool
	}{
		{
			name: "single static unit",
			constructs: []core.Construct{
				su,
			},
			constructIdToResources: map[string][]core.Resource{
				su.Id(): {
					resources.NewS3Bucket(su, appName),
				},
			},
			cfg: config.Application{
				StaticUnit: map[string]*config.StaticUnit{
					su.ID: {
						ContentDeliveryNetwork: config.ContentDeliveryNetwork{Id: cdn1},
					},
				},
				AppName: appName,
			},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:cloudfront_distribution:test-cdn",
					"aws:cloudfront_origin_access_identity:test-su-su",
					"aws:s3_bucket:test-su",
					"aws:s3_bucket_policy:test-su-su",
				},
				Deps: []graph.Edge[string]{
					{Source: "aws:cloudfront_distribution:test-cdn", Destination: "aws:cloudfront_origin_access_identity:test-su-su"},
					{Source: "aws:cloudfront_distribution:test-cdn", Destination: "aws:s3_bucket:test-su"},
					{Source: "aws:s3_bucket_policy:test-su-su", Destination: "aws:cloudfront_origin_access_identity:test-su-su"},
					{Source: "aws:s3_bucket_policy:test-su-su", Destination: "aws:s3_bucket:test-su"},
				},
			},
			wantRefs: map[core.ResourceId][]core.AnnotationKey{
				distro.Id(): {su.Provenance()},
			},
		},
		{
			name: "single expose",
			constructs: []core.Construct{
				gw,
			},
			constructIdToResources: map[string][]core.Resource{
				gw.Id(): {
					resources.NewApiStage(resources.NewApiDeployment(resources.NewRestApi(appName, gw), nil, nil), "stage", nil),
				},
			},
			cfg: config.Application{
				Exposed: map[string]*config.Expose{
					gw.ID: {
						ContentDeliveryNetwork: config.ContentDeliveryNetwork{Id: cdn1},
					},
				},
				AppName: appName,
			},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:api_stage:test-gw-stage",
					"aws:cloudfront_distribution:test-cdn",
				},
				Deps: []graph.Edge[string]{
					{Source: "aws:cloudfront_distribution:test-cdn", Destination: "aws:api_stage:test-gw-stage"},
				},
			},
			wantRefs: map[core.ResourceId][]core.AnnotationKey{
				distro.Id(): {gw.Provenance()},
			},
		},
		{
			name: "multiple cdns",
			constructs: []core.Construct{
				gw, su,
			},
			constructIdToResources: map[string][]core.Resource{
				gw.Id(): {
					resources.NewApiStage(resources.NewApiDeployment(resources.NewRestApi(appName, gw), nil, nil), "stage", nil),
				},
				su.Id(): {
					resources.NewS3Bucket(su, appName),
				},
			},
			cfg: config.Application{
				StaticUnit: map[string]*config.StaticUnit{
					su.ID: {
						ContentDeliveryNetwork: config.ContentDeliveryNetwork{Id: cdn1},
					},
				},
				Exposed: map[string]*config.Expose{
					gw.ID: {
						ContentDeliveryNetwork: config.ContentDeliveryNetwork{Id: cdn2},
					},
				},
				AppName: appName,
			},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:api_stage:test-gw-stage",
					"aws:cloudfront_distribution:test-cdn",
					"aws:cloudfront_distribution:test-otherCdn",
					"aws:cloudfront_origin_access_identity:test-su-su",
					"aws:s3_bucket:test-su",
					"aws:s3_bucket_policy:test-su-su",
				},
				Deps: []graph.Edge[string]{
					{Source: "aws:cloudfront_distribution:test-cdn", Destination: "aws:cloudfront_origin_access_identity:test-su-su"},
					{Source: "aws:cloudfront_distribution:test-cdn", Destination: "aws:s3_bucket:test-su"},
					{Source: "aws:cloudfront_distribution:test-otherCdn", Destination: "aws:api_stage:test-gw-stage"},
					{Source: "aws:s3_bucket_policy:test-su-su", Destination: "aws:cloudfront_origin_access_identity:test-su-su"},
					{Source: "aws:s3_bucket_policy:test-su-su", Destination: "aws:s3_bucket:test-su"},
				},
			},
			wantRefs: map[core.ResourceId][]core.AnnotationKey{
				distro.Id():  {su.Provenance()},
				distro2.Id(): {gw.Provenance()},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			aws := AWS{
				Config:                 &tt.cfg,
				constructIdToResources: tt.constructIdToResources,
			}
			dag := core.NewResourceGraph()

			for _, resources := range tt.constructIdToResources {
				for _, res := range resources {
					dag.AddResource(res)
				}
			}
			result := core.NewConstructGraph()
			for _, unit := range tt.constructs {
				result.AddConstruct(unit)
			}

			err := aws.createCDNs(result, dag)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}

			tt.want.Assert(t, dag)

			for key, val := range tt.wantRefs {
				assert.ElementsMatch(val, dag.GetResource(key).KlothoConstructRef())
			}
		})

	}
}
