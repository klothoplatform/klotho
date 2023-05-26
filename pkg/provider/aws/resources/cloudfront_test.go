package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_CloudfrontDistributionCreate(t *testing.T) {
	eu := &core.StaticUnit{AnnotationKey: core.AnnotationKey{ID: "test"}}
	cases := []coretesting.CreateCase[CloudfrontDistributionCreateParams, *CloudfrontDistribution]{
		{
			Name: "new",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					`aws:cloudfront_distribution:my-app-some_cdn`,
				},
				Deps: nil,
			},
			Check: func(assert *assert.Assertions, distro *CloudfrontDistribution) {
				assert.NotNil(distro.Restrictions)
				assert.NotNil(distro.DefaultCacheBehavior)
			},
		},
		{
			Name:     "already exists",
			Existing: &CloudfrontDistribution{Name: "my-app-some_cdn"},
			WantErr:  true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = CloudfrontDistributionCreateParams{
				CdnId:   "some_cdn",
				AppName: "my-app",
				Refs:    core.AnnotationKeySetOf(eu.AnnotationKey),
			}
			tt.Run(t)
		})
	}

}
