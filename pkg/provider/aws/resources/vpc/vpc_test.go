package vpc

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/stretchr/testify/assert"
)

func Test_NewVpc(t *testing.T) {
	assert := assert.New(t)
	vpc := NewVpc("test-app")
	assert.Equal(vpc.Name, "test_app")
	assert.Nil(vpc.ConstructsRef)
	assert.Equal(vpc.CidrBlock, "10.0.0.0/16")
	assert.Equal(vpc.enableDnsSupport, true)
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
