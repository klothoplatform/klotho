package vpc

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/stretchr/testify/assert"
)

func Test_NewInternetGateway(t *testing.T) {
	assert := assert.New(t)
	vpc := NewVpc("test-app")
	igw := NewInternetGateway("test-app", "igw", vpc)
	assert.Equal(igw.Name, "test_app_igw")
	assert.Nil(igw.ConstructsRef)
	assert.Equal(igw.Vpc, vpc)
}

func Test_InternetGatewayProvider(t *testing.T) {
	assert := assert.New(t)
	vpc := NewVpc("test-app")
	igw := NewInternetGateway("test-app", "igw", vpc)
	assert.Equal(igw.Provider(), resources.AWS_PROVIDER)
}

func Test_InternetGatewayId(t *testing.T) {
	assert := assert.New(t)
	vpc := NewVpc("test-app")
	igw := NewInternetGateway("test-app", "igw", vpc)
	assert.Equal(igw.Id(), "aws:internet_gateway:test_app_igw")
}

func Test_InternetGatewayKlothoConstructRef(t *testing.T) {
	assert := assert.New(t)
	vpc := NewVpc("test-app")
	igw := NewInternetGateway("test-app", "igw", vpc)
	assert.Nil(igw.ConstructsRef)
}
