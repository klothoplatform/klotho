package vpc

import (
	"testing"

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
