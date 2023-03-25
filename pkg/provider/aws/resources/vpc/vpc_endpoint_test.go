package vpc

import (
	"testing"

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
