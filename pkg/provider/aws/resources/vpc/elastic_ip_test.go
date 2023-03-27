package vpc

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/stretchr/testify/assert"
)

func Test_NewElasticIp(t *testing.T) {
	assert := assert.New(t)
	ip := NewElasticIp("test-app", "ip1")
	assert.Equal(ip.Name, "test_app_ip1")
	assert.Nil(ip.ConstructsRef)
}

func Test_ElasticIpProvider(t *testing.T) {
	assert := assert.New(t)
	ip := NewElasticIp("test-app", "ip1")
	assert.Equal(ip.Provider(), resources.AWS_PROVIDER)
}

func Test_ElasticIpId(t *testing.T) {
	assert := assert.New(t)
	ip := NewElasticIp("test-app", "ip1")
	assert.Equal(ip.Id(), "aws:elastic_ip:test_app_ip1")
}

func Test_ElasticIpKlothoConstructRef(t *testing.T) {
	assert := assert.New(t)
	ip := NewElasticIp("test-app", "ip1")
	assert.Nil(ip.ConstructsRef)
}
