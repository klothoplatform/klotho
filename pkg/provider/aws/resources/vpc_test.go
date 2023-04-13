package resources

import (
	"fmt"
	"testing"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_CreateNetwork(t *testing.T) {
	appName := "test-app"
	cases := []struct {
		name              string
		existingResources []core.Resource
		want              coretesting.ResourcesExpectation
	}{
		{
			name: "happy path",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:elastic_ip:test_app_private1",
					"aws:elastic_ip:test_app_private2",
					"aws:internet_gateway:test_app_igw1",
					"aws:nat_gateway:test_app_private1",
					"aws:nat_gateway:test_app_private2",
					"aws:region:region",
					"aws:vpc:test_app",
					"aws:vpc_endpoint:test_app_dynamodb",
					"aws:vpc_endpoint:test_app_lambda",
					"aws:vpc_endpoint:test_app_s3",
					"aws:vpc_endpoint:test_app_secretsmanager",
					"aws:vpc_endpoint:test_app_sns",
					"aws:vpc_endpoint:test_app_sqs",
					"aws:vpc_subnet:test_app_private1",
					"aws:vpc_subnet:test_app_private2",
					"aws:vpc_subnet:test_app_public1",
					"aws:vpc_subnet:test_app_public2",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:availability_zones:AvailabilityZones", Destination: "aws:region:region"},
					{Source: "aws:internet_gateway:test_app_igw1", Destination: "aws:vpc:test_app"},
					{Source: "aws:nat_gateway:test_app_private1", Destination: "aws:elastic_ip:test_app_private1"},
					{Source: "aws:nat_gateway:test_app_private1", Destination: "aws:vpc_subnet:test_app_private1"},
					{Source: "aws:nat_gateway:test_app_private2", Destination: "aws:elastic_ip:test_app_private2"},
					{Source: "aws:nat_gateway:test_app_private2", Destination: "aws:vpc_subnet:test_app_private2"},
					{Source: "aws:vpc_endpoint:test_app_dynamodb", Destination: "aws:region:region"},
					{Source: "aws:vpc_endpoint:test_app_dynamodb", Destination: "aws:vpc:test_app"},
					{Source: "aws:vpc_endpoint:test_app_lambda", Destination: "aws:region:region"},
					{Source: "aws:vpc_endpoint:test_app_lambda", Destination: "aws:vpc:test_app"},
					{Source: "aws:vpc_endpoint:test_app_lambda", Destination: "aws:vpc_subnet:test_app_private1"},
					{Source: "aws:vpc_endpoint:test_app_lambda", Destination: "aws:vpc_subnet:test_app_private2"},
					{Source: "aws:vpc_endpoint:test_app_s3", Destination: "aws:region:region"},
					{Source: "aws:vpc_endpoint:test_app_s3", Destination: "aws:vpc:test_app"},
					{Source: "aws:vpc_endpoint:test_app_secretsmanager", Destination: "aws:region:region"},
					{Source: "aws:vpc_endpoint:test_app_secretsmanager", Destination: "aws:vpc:test_app"},
					{Source: "aws:vpc_endpoint:test_app_secretsmanager", Destination: "aws:vpc_subnet:test_app_private1"},
					{Source: "aws:vpc_endpoint:test_app_secretsmanager", Destination: "aws:vpc_subnet:test_app_private2"},
					{Source: "aws:vpc_endpoint:test_app_sns", Destination: "aws:region:region"},
					{Source: "aws:vpc_endpoint:test_app_sns", Destination: "aws:vpc:test_app"},
					{Source: "aws:vpc_endpoint:test_app_sns", Destination: "aws:vpc_subnet:test_app_private1"},
					{Source: "aws:vpc_endpoint:test_app_sns", Destination: "aws:vpc_subnet:test_app_private2"},
					{Source: "aws:vpc_endpoint:test_app_sqs", Destination: "aws:region:region"},
					{Source: "aws:vpc_endpoint:test_app_sqs", Destination: "aws:vpc:test_app"},
					{Source: "aws:vpc_endpoint:test_app_sqs", Destination: "aws:vpc_subnet:test_app_private1"},
					{Source: "aws:vpc_endpoint:test_app_sqs", Destination: "aws:vpc_subnet:test_app_private2"},
					{Source: "aws:vpc_subnet:test_app_private1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:vpc_subnet:test_app_private1", Destination: "aws:vpc:test_app"},
					{Source: "aws:vpc_subnet:test_app_private2", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:vpc_subnet:test_app_private2", Destination: "aws:vpc:test_app"},
					{Source: "aws:vpc_subnet:test_app_public1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:vpc_subnet:test_app_public1", Destination: "aws:vpc:test_app"},
					{Source: "aws:vpc_subnet:test_app_public2", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:vpc_subnet:test_app_public2", Destination: "aws:vpc:test_app"},
				},
			},
		},
		{
			name:              "happy path",
			existingResources: []core.Resource{NewVpc(appName)},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{"aws:vpc:test_app"},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			for _, res := range tt.existingResources {
				dag.AddResource(res)
			}

			cfg := &config.Application{
				AppName: appName,
			}
			result := CreateNetwork(cfg, dag)
			assert.NotNil(result)

			tt.want.Assert(t, dag)
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
					CreatePublicSubnet(fmt.Sprintf("public%d", i), core.IaCValue{Resource: NewAvailabilityZones()}, vpc, spec.Cidr, dag)
				} else {
					CreatePrivateSubnet("test-app", fmt.Sprintf("private%d", i), core.IaCValue{Resource: NewAvailabilityZones()}, vpc, spec.Cidr, dag)
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

func Test_GetPrivateSubnets(t *testing.T) {

	type subnetSpec struct {
		Cidr   string
		Public bool
	}

	cases := []struct {
		name    string
		subnets []subnetSpec
		want    []subnetSpec
	}{
		{
			name: "happy path",
			subnets: []subnetSpec{
				{"10.0.1.0/24", false},
				{"10.0.2.0/24", false},
				{"10.0.3.0/24", true},
				{"10.0.4.0/24", true},
			},
			want: []subnetSpec{
				{"10.0.1.0/24", false}, {"10.0.2.0/24", false},
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
					CreatePublicSubnet(fmt.Sprintf("public%d", i), core.IaCValue{Resource: NewAvailabilityZones()}, vpc, spec.Cidr, dag)
				} else {
					CreatePrivateSubnet("test-app", fmt.Sprintf("private%d", i), core.IaCValue{Resource: NewAvailabilityZones()}, vpc, spec.Cidr, dag)
				}
			}

			result := vpc.GetPrivateSubnets(dag)
			var got []subnetSpec
			for _, sn := range result {
				got = append(got, subnetSpec{Cidr: sn.CidrBlock, Public: sn.Type == PublicSubnet})
			}
			assert.ElementsMatch(got, tt.want)
		})
	}
}

func Test_CreatePrivateSubnet(t *testing.T) {
	appName := "test-app"
	cases := []struct {
		name string
		want coretesting.ResourcesExpectation
	}{
		{
			name: "happy path",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:vpc:test_app",
					"aws:availability_zones:AvailabilityZones",
					"aws:vpc_subnet:test_app_private1",
					"aws:elastic_ip:test_app_private1",
					"aws:nat_gateway:test_app_private1",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:vpc_subnet:test_app_private1", Destination: "aws:vpc:test_app"},
					{Source: "aws:vpc_subnet:test_app_private1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:nat_gateway:test_app_private1", Destination: "aws:vpc_subnet:test_app_private1"},
					{Source: "aws:nat_gateway:test_app_private1", Destination: "aws:elastic_ip:test_app_private1"},
				},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			dag := core.NewResourceGraph()
			vpc := NewVpc(appName)
			az := NewAvailabilityZones()
			CreatePrivateSubnet(appName, "private1", core.IaCValue{Resource: az}, vpc, "0", dag)

			tt.want.Assert(t, dag)
		})
	}
}

func Test_CreatePublicSubnet(t *testing.T) {
	appName := "test-app"
	cases := []struct {
		name string
		want coretesting.ResourcesExpectation
	}{
		{
			name: "happy path",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{"aws:vpc:test_app", "aws:availability_zones:AvailabilityZones", "aws:vpc_subnet:test_app_public1"},
				Deps: []coretesting.StringDep{
					{Source: "aws:vpc_subnet:test_app_public1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:vpc_subnet:test_app_public1", Destination: "aws:vpc:test_app"},
				},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			dag := core.NewResourceGraph()
			vpc := NewVpc(appName)
			az := NewAvailabilityZones()
			CreatePublicSubnet("public1", core.IaCValue{Resource: az}, vpc, "0", dag)

			tt.want.Assert(t, dag)
		})
	}
}

func Test_CreateGatewayVpcEndpoint(t *testing.T) {
	appName := "test-app"
	cases := []struct {
		name string
		want coretesting.ResourcesExpectation
	}{
		{
			name: "happy path",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{"aws:vpc:test_app", "aws:region:region", "aws:vpc_endpoint:test_app_s3", "aws:vpc_subnet:test_app_1", "aws:vpc_subnet:test_app_2"},
				Deps: []coretesting.StringDep{
					{Source: "aws:vpc_endpoint:test_app_s3", Destination: "aws:vpc:test_app"},
					{Source: "aws:vpc_endpoint:test_app_s3", Destination: "aws:region:region"},
					{Source: "aws:vpc_subnet:test_app_1", Destination: "aws:vpc:test_app"},
					{Source: "aws:vpc_subnet:test_app_2", Destination: "aws:vpc:test_app"},
				},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			dag := core.NewResourceGraph()
			vpc := NewVpc(appName)
			subnet1 := NewSubnet("1", vpc, "", PrivateSubnet, core.IaCValue{})
			subnet2 := NewSubnet("2", vpc, "", PrivateSubnet, core.IaCValue{})
			region := NewRegion()
			dag.AddDependency(subnet1, vpc)
			dag.AddDependency(subnet2, vpc)
			CreateGatewayVpcEndpoint("s3", vpc, region, dag)
			tt.want.Assert(t, dag)
		})
	}
}

func Test_CreateInterfaceVpcEndpoint(t *testing.T) {
	appName := "test-app"
	cases := []struct {
		name string
		want coretesting.ResourcesExpectation
	}{
		{
			name: "happy path",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{"aws:vpc:test_app", "aws:region:region", "aws:vpc_endpoint:test_app_s3", "aws:vpc_subnet:test_app_1", "aws:vpc_subnet:test_app_2"},
				Deps: []coretesting.StringDep{
					{Destination: "aws:vpc:test_app", Source: "aws:vpc_endpoint:test_app_s3"},
					{Destination: "aws:region:region", Source: "aws:vpc_endpoint:test_app_s3"},
					{Destination: "aws:vpc:test_app", Source: "aws:vpc_subnet:test_app_1"},
					{Destination: "aws:vpc:test_app", Source: "aws:vpc_subnet:test_app_2"},
					{Destination: "aws:vpc_subnet:test_app_1", Source: "aws:vpc_endpoint:test_app_s3"},
					{Destination: "aws:vpc_subnet:test_app_2", Source: "aws:vpc_endpoint:test_app_s3"},
				},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			dag := core.NewResourceGraph()
			vpc := NewVpc(appName)
			subnet1 := NewSubnet("1", vpc, "", PrivateSubnet, core.IaCValue{})
			subnet2 := NewSubnet("2", vpc, "", PrivateSubnet, core.IaCValue{})
			region := NewRegion()
			dag.AddDependency(subnet1, vpc)
			dag.AddDependency(subnet2, vpc)
			CreateInterfaceVpcEndpoint("s3", vpc, region, dag)
			tt.want.Assert(t, dag)
		})
	}
}
