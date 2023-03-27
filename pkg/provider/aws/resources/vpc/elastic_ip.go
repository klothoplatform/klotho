package vpc

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

const ELASTIC_IP_TYPE = "elastic_ip"

var elasticIpSanitizer = aws.SubnetSanitizer

type (
	ElasticIp struct {
		Name          string
		ConstructsRef []core.AnnotationKey
	}
)

func NewElasticIp(appName string, ipName string) *ElasticIp {
	return &ElasticIp{
		Name: elasticIpSanitizer.Apply(fmt.Sprintf("%s-%s", appName, ipName)),
	}
}

// Provider returns name of the provider the resource is correlated to
func (subnet *ElasticIp) Provider() string {
	return resources.AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (subnet *ElasticIp) KlothoConstructRef() []core.AnnotationKey {
	return subnet.ConstructsRef
}

// ID returns the id of the cloud resource
func (subnet *ElasticIp) Id() string {
	return fmt.Sprintf("%s:%s:%s", subnet.Provider(), ELASTIC_IP_TYPE, subnet.Name)
}
