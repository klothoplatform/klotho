package vpc

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
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
	assert.Equal(vpc.Provider(), resources.AWS_PROVIDER)
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
					{source: "aws:vpc:test_app", dest: "aws:internet_gateway:test_app_igw1"},
					{source: "aws:vpc:test_app", dest: "aws:vpc_subnet:test_app_private1"},
					{source: "aws:vpc:test_app", dest: "aws:vpc_subnet:test_app_private2"},
					{source: "aws:vpc:test_app", dest: "aws:vpc_subnet:test_app_public1"},
					{source: "aws:vpc:test_app", dest: "aws:vpc_subnet:test_app_public2"},
					{source: "aws:vpc:test_app", dest: "aws:vpc_endpoint:test_app_sqs"},
					{source: "aws:vpc:test_app", dest: "aws:vpc_endpoint:test_app_s3"},
					{source: "aws:vpc:test_app", dest: "aws:vpc_endpoint:test_app_dynamodb"},
					{source: "aws:vpc:test_app", dest: "aws:vpc_endpoint:test_app_sns"},
					{source: "aws:vpc:test_app", dest: "aws:vpc_endpoint:test_app_lambda"},
					{source: "aws:vpc:test_app", dest: "aws:vpc_endpoint:test_app_secretsmanager"},
					{source: "aws:region:region", dest: "aws:vpc_endpoint:test_app_sqs"},
					{source: "aws:region:region", dest: "aws:vpc_endpoint:test_app_s3"},
					{source: "aws:region:region", dest: "aws:vpc_endpoint:test_app_dynamodb"},
					{source: "aws:region:region", dest: "aws:vpc_endpoint:test_app_sns"},
					{source: "aws:region:region", dest: "aws:vpc_endpoint:test_app_lambda"},
					{source: "aws:region:region", dest: "aws:vpc_endpoint:test_app_secretsmanager"},
					{source: "aws:vpc_subnet:test_app_private2", dest: "aws:nat_gateway:test_app_private2"},
					{source: "aws:elastic_ip:test_app_private2", dest: "aws:nat_gateway:test_app_private2"},
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
	vpc := NewVpc("test-app")
	subnet1 := NewSubnet("private1", vpc, "10.0.0.0/24", PrivateSubnet)
	subnet2 := NewSubnet("private2", vpc, "10.0.0.0/24", PrivateSubnet)
	subnet3 := NewSubnet("private3", vpc, "10.0.0.0/24", PrivateSubnet)
	subnet4 := NewSubnet("private4", vpc, "10.0.0.0/24", PrivateSubnet)

	cases := []struct {
		name    string
		vpc     *Vpc
		subnets []*Subnet
		want    []*Subnet
	}{
		{
			name:    "happy path",
			vpc:     vpc,
			subnets: []*Subnet{subnet1, subnet2, subnet3, subnet4},
			want:    []*Subnet{subnet1, subnet2, subnet3, subnet4},
		},
		{
			name: "no subnets",
			vpc:  vpc,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()

			dag.AddResource(tt.vpc)
			for _, s := range tt.subnets {
				dag.AddResource(s)
				dag.AddDependency(tt.vpc, s)
			}
			result := tt.vpc.GetVpcSubnets(dag)
			assert.ElementsMatch(result, tt.want)
		})
	}
}
