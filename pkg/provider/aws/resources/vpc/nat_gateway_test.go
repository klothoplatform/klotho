package vpc

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/stretchr/testify/assert"
)

func Test_NewNatGateway(t *testing.T) {
	assert := assert.New(t)
	vpc := NewVpc("test-app")
	ip := NewElasticIp("test-app", "ip1")
	subnet := NewSubnet("private1", vpc, "10.0.0.0/24", PrivateSubnet)
	natGateway := NewNatGateway("test-app", "natgw1", subnet, ip)
	assert.Equal(natGateway.Name, "test_app_natgw1")
	assert.Nil(natGateway.ConstructsRef)
	assert.Equal(natGateway.Subnet, subnet)
	assert.Equal(natGateway.ElasticIp, ip)
}

func Test_NatGatewayProvider(t *testing.T) {
	assert := assert.New(t)
	vpc := NewVpc("test-app")
	ip := NewElasticIp("test-app", "ip1")
	subnet := NewSubnet("private1", vpc, "10.0.0.0/24", PrivateSubnet)
	natGateway := NewNatGateway("test-app", "natgw1", subnet, ip)
	assert.Equal(natGateway.Provider(), resources.AWS_PROVIDER)
}

func Test_NatGatewayId(t *testing.T) {
	assert := assert.New(t)
	vpc := NewVpc("test-app")
	ip := NewElasticIp("test-app", "ip1")
	subnet := NewSubnet("private1", vpc, "10.0.0.0/24", PrivateSubnet)
	natGateway := NewNatGateway("test-app", "natgw1", subnet, ip)
	assert.Equal(natGateway.Id(), "aws:nat_gateway:test_app_natgw1")
}

func Test_NatGatewayKlothoConstructRef(t *testing.T) {
	assert := assert.New(t)
	vpc := NewVpc("test-app")
	ip := NewElasticIp("test-app", "ip1")
	subnet := NewSubnet("private1", vpc, "10.0.0.0/24", PrivateSubnet)
	natGateway := NewNatGateway("test-app", "natgw1", subnet, ip)
	assert.Nil(natGateway.ConstructsRef)
}
