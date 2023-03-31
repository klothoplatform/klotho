package resources

import (
	"fmt"
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/stretchr/testify/assert"
)

func Test_NewVpc(t *testing.T) {
	assert := assert.New(t)
	vpc := NewVpc("test-app")
	assert.Equal(vpc.Name, "test_app")
	assert.Nil(vpc.ConstructsRef)
	assert.Equal(vpc.CidrBlock, "10.0.0.0/16")
	assert.Equal(vpc.EnableDnsSupport, true)
}

func Test_VpcProvider(t *testing.T) {
	assert := assert.New(t)
	vpc := NewVpc("test-app")
	assert.Equal(vpc.Provider(), AWS_PROVIDER)
}

func Test_VpcId(t *testing.T) {
	assert := assert.New(t)
	vpc := NewVpc("test-app")
	assert.Equal(vpc.Id(), "aws:vpc:test_app")
}

func Test_VpcKlothoConstructRef(t *testing.T) {
	assert := assert.New(t)
	vpc := NewVpc("test-app")
	assert.Nil(vpc.ConstructsRef)
}

func Test_CreateNetwork(t *testing.T) {
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
				nodes: []string{"aws:vpc:test_app", "aws:vpc_subnet:test_app_private1", "aws:vpc_subnet:test_app_private2", "aws:vpc_subnet:test_app_public1", "aws:vpc_subnet:test_app_public2",
					"aws:vpc_endpoint:test_app_s3", "aws:vpc_endpoint:test_app_sqs", "aws:vpc_endpoint:test_app_sns", "aws:vpc_endpoint:test_app_lambda",
					"aws:vpc_endpoint:test_app_secretsmanager", "aws:vpc_endpoint:test_app_dynamodb", "aws:elastic_ip:test_app_private2", "aws:elastic_ip:test_app_private1",
					"aws:nat_gateway:test_app_private1", "aws:nat_gateway:test_app_private2", "aws:region:region", "aws:internet_gateway:test_app_igw1",
				},
				deps: []stringDep{
					{source: "aws:internet_gateway:test_app_igw1", dest: "aws:vpc:test_app"},
					{source: "aws:vpc_subnet:test_app_private1", dest: "aws:vpc:test_app"},
					{source: "aws:vpc_subnet:test_app_private2", dest: "aws:vpc:test_app"},
					{source: "aws:vpc_subnet:test_app_public1", dest: "aws:vpc:test_app"},
					{source: "aws:vpc_subnet:test_app_public2", dest: "aws:vpc:test_app"},
					{source: "aws:vpc_endpoint:test_app_sqs", dest: "aws:vpc:test_app"},
					{source: "aws:vpc_endpoint:test_app_s3", dest: "aws:vpc:test_app"},
					{source: "aws:vpc_endpoint:test_app_dynamodb", dest: "aws:vpc:test_app"},
					{source: "aws:vpc_endpoint:test_app_sns", dest: "aws:vpc:test_app"},
					{source: "aws:vpc_endpoint:test_app_lambda", dest: "aws:vpc:test_app"},
					{source: "aws:vpc_endpoint:test_app_secretsmanager", dest: "aws:vpc:test_app"},
					{source: "aws:vpc_endpoint:test_app_sqs", dest: "aws:region:region"},
					{source: "aws:vpc_endpoint:test_app_s3", dest: "aws:region:region"},
					{source: "aws:vpc_endpoint:test_app_dynamodb", dest: "aws:region:region"},
					{source: "aws:vpc_endpoint:test_app_sns", dest: "aws:region:region"},
					{source: "aws:vpc_endpoint:test_app_lambda", dest: "aws:region:region"},
					{source: "aws:vpc_endpoint:test_app_secretsmanager", dest: "aws:region:region"},
					{source: "aws:nat_gateway:test_app_private2", dest: "aws:vpc_subnet:test_app_private2"},
					{source: "aws:nat_gateway:test_app_private2", dest: "aws:elastic_ip:test_app_private2"},
					{source: "aws:nat_gateway:test_app_private1", dest: "aws:vpc_subnet:test_app_private1"},
					{source: "aws:nat_gateway:test_app_private1", dest: "aws:elastic_ip:test_app_private1"},
				},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			CreateNetwork(appName, dag)
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

func Test_GetVpcSubnets(t *testing.T) {

	type subnetSpec struct {
		Cidr   string
		Public bool
	}

	cases := []struct {
		name    string
		subnets []subnetSpec
	}{
		{
			name: "happy path",
			subnets: []subnetSpec{
				{"10.0.1.0/24", false},
				{"10.0.2.0/24", false},
				{"10.0.3.0/24", true},
				{"10.0.4.0/24", true},
			},
		},
		{
			name: "no subnets",
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()

			vpc := NewVpc("test-app")
			dag.AddResource(vpc)
			for i, spec := range tt.subnets {
				if spec.Public {
					CreatePublicSubnet(fmt.Sprintf("public%d", i), vpc, spec.Cidr, dag)
				} else {
					CreatePrivateSubnet("test-app", fmt.Sprintf("private%d", i), vpc, spec.Cidr, dag)
				}
			}

			result := vpc.GetVpcSubnets(dag)
			var got []subnetSpec
			for _, sn := range result {
				got = append(got, subnetSpec{Cidr: sn.CidrBlock, Public: sn.Type == PublicSubnet})
			}
			assert.ElementsMatch(got, tt.subnets)
		})
	}
}
