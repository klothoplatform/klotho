package vpc

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/stretchr/testify/assert"
)

func Test_NewVpcEndpoint(t *testing.T) {
	assert := assert.New(t)
	vpc := NewVpc("test-app")
	region := resources.NewRegion()
	vpce := NewVpcEndpoint("s3", vpc, "Interface", region)
	assert.Equal(vpce.Name, "test_app_s3")
	assert.Nil(vpce.ConstructsRef)
	assert.Equal(vpce.ServiceName, "s3")
	assert.Equal(vpce.Vpc, vpc)
	assert.Equal(vpce.VpcEndpointType, "Interface")

}

func Test_VpcEndpointProvider(t *testing.T) {
	assert := assert.New(t)
	vpc := NewVpc("test-app")
	region := resources.NewRegion()
	vpce := NewVpcEndpoint("s3", vpc, "Interface", region)
	assert.Equal(vpce.Provider(), resources.AWS_PROVIDER)
}

func Test_VpcEndpointId(t *testing.T) {
	assert := assert.New(t)
	vpc := NewVpc("test-app")
	region := resources.NewRegion()
	vpce := NewVpcEndpoint("s3", vpc, "Interface", region)
	assert.Equal(vpce.Id(), "aws:vpc_endpoint:test_app_s3")
}

func Test_VpcEndpointKlothoConstructRef(t *testing.T) {
	assert := assert.New(t)
	vpc := NewVpc("test-app")
	region := resources.NewRegion()
	vpce := NewVpcEndpoint("s3", vpc, "Interface", region)
	assert.Nil(vpce.ConstructsRef)
}

func Test_CreateGatewayVpcEndpoint(t *testing.T) {
	appName := "test-app"
	type stringDep struct {
		source string
		dest   string
	}
	type testResult struct {
		nodes []string
		deps  []stringDep
	}
	cases := []struct {
		name string
		want testResult
	}{
		{
			name: "happy path",
			want: testResult{
				nodes: []string{"aws:vpc:test_app", "aws:region:region", "aws:vpc_endpoint:test_app_s3"},
				deps: []stringDep{
					{source: "aws:vpc:test_app", dest: "aws:vpc_endpoint:test_app_s3"},
					{source: "aws:region:region", dest: "aws:vpc_endpoint:test_app_s3"},
				},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			vpc := NewVpc(appName)
			region := resources.NewRegion()
			dag.AddResource(vpc)
			dag.AddResource(region)
			CreateGatewayVpcEndpoint("s3", vpc, region, dag)
			for _, id := range tt.want.nodes {
				assert.NotNil(dag.GetResource(id))
			}
			for _, dep := range tt.want.deps {
				assert.NotNil(dag.GetDependency(dep.source, dep.dest))
			}
			assert.Len(dag.ListResources(), len(tt.want.nodes))
			assert.Len(dag.ListDependencies(), len(tt.want.deps))
		})
	}
}

func Test_CreateInterfaceVpcEndpoint(t *testing.T) {
	appName := "test-app"
	type stringDep struct {
		source string
		dest   string
	}
	type testResult struct {
		nodes []string
		deps  []stringDep
	}
	cases := []struct {
		name string
		want testResult
	}{
		{
			name: "happy path",
			want: testResult{
				nodes: []string{"aws:vpc:test_app", "aws:region:region", "aws:vpc_endpoint:test_app_s3"},
				deps: []stringDep{
					{source: "aws:vpc:test_app", dest: "aws:vpc_endpoint:test_app_s3"},
					{source: "aws:region:region", dest: "aws:vpc_endpoint:test_app_s3"},
				},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			vpc := NewVpc(appName)
			region := resources.NewRegion()
			dag.AddResource(vpc)
			dag.AddResource(region)
			CreateGatewayVpcEndpoint("s3", vpc, region, dag)
			for _, id := range tt.want.nodes {
				assert.NotNil(dag.GetResource(id))
			}
			for _, dep := range tt.want.deps {
				assert.NotNil(dag.GetDependency(dep.source, dep.dest))
			}
			assert.Len(dag.ListResources(), len(tt.want.nodes))
			assert.Len(dag.ListDependencies(), len(tt.want.deps))
		})
	}
}
