package vpc

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/stretchr/testify/assert"
)

func Test_NewSubnet(t *testing.T) {
	assert := assert.New(t)
	vpc := NewVpc("test-app")
	subnet := NewSubnet("private1", vpc, "10.0.0.0/24", PrivateSubnet)
	assert.Equal(subnet.Name, "test_app_private1")
	assert.Nil(subnet.ConstructsRef)
	assert.Equal(subnet.CidrBlock, "10.0.0.0/24")
	assert.Equal(subnet.Vpc, vpc)
	assert.Equal(subnet.Type, "private")
}

func Test_SubnetProvider(t *testing.T) {
	assert := assert.New(t)
	vpc := NewVpc("test-app")
	subnet := NewSubnet("private1", vpc, "10.0.0.0/24", PrivateSubnet)
	assert.Equal(subnet.Provider(), resources.AWS_PROVIDER)
}

func Test_SubnetId(t *testing.T) {
	assert := assert.New(t)
	vpc := NewVpc("test-app")
	subnet := NewSubnet("private1", vpc, "10.0.0.0/24", PrivateSubnet)
	assert.Equal(subnet.Id(), "aws:vpc_subnet:test_app_private1")
}

func Test_SubnetKlothoConstructRef(t *testing.T) {
	assert := assert.New(t)
	vpc := NewVpc("test-app")
	subnet := NewSubnet("private1", vpc, "10.0.0.0/24", PrivateSubnet)
	assert.Nil(subnet.ConstructsRef)
}

func Test_CreatePrivateSubnet(t *testing.T) {
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
				nodes: []string{"aws:vpc:test_app", "aws:vpc_subnet:test_app_private1", "aws:elastic_ip:test_app_private1", "aws:nat_gateway:test_app_private1"},
				deps: []stringDep{
					{source: "aws:vpc:test_app", dest: "aws:vpc_subnet:test_app_private1"},
					{source: "aws:vpc_subnet:test_app_private1", dest: "aws:nat_gateway:test_app_private1"},
					{source: "aws:elastic_ip:test_app_private1", dest: "aws:nat_gateway:test_app_private1"},
				},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			vpc := NewVpc(appName)
			dag.AddResource(vpc)
			CreatePrivateSubnet(appName, "private1", vpc, "0", dag)
			for _, id := range tt.want.nodes {
				assert.NotNil(dag.GetResource(id))
			}
			for _, dep := range tt.want.deps {
				assert.NotNil(dag.GetDependency(dep.source, dep.dest))
			}
			assert.Len(dag.ListConstructs(), len(tt.want.nodes))
			assert.Len(dag.ListDependencies(), len(tt.want.deps))
		})
	}
}

func Test_CreatePublicSubnet(t *testing.T) {
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
				nodes: []string{"aws:vpc:test_app", "aws:vpc_subnet:test_app_public1"},
				deps: []stringDep{
					{source: "aws:vpc:test_app", dest: "aws:vpc_subnet:test_app_public1"},
				},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			vpc := NewVpc(appName)
			dag.AddResource(vpc)
			CreatePublicSubnet("public1", vpc, "0", dag)
			for _, id := range tt.want.nodes {
				assert.NotNil(dag.GetResource(id))
			}
			for _, dep := range tt.want.deps {
				assert.NotNil(dag.GetDependency(dep.source, dep.dest))
			}
			assert.Len(dag.ListConstructs(), len(tt.want.nodes))
			assert.Len(dag.ListDependencies(), len(tt.want.deps))
		})
	}
}
