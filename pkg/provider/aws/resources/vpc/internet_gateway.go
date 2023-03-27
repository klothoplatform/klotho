package vpc

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

const INTERNET_GATEWAY_TYPE = "internet_gateway"

var igwSanitizer = aws.SubnetSanitizer

type (
	InternetGateway struct {
		Name          string
		ConstructsRef []core.AnnotationKey
		Vpc           *Vpc
	}
)

func NewInternetGateway(appName string, igwName string, vpc *Vpc) *InternetGateway {
	return &InternetGateway{
		Name: igwSanitizer.Apply(fmt.Sprintf("%s-%s", appName, igwName)),
		Vpc:  vpc,
	}
}

// Provider returns name of the provider the resource is correlated to
func (igw *InternetGateway) Provider() string {
	return resources.AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (igw *InternetGateway) KlothoConstructRef() []core.AnnotationKey {
	return igw.ConstructsRef
}

// ID returns the id of the cloud resource
func (igw *InternetGateway) Id() string {
	return fmt.Sprintf("%s:%s:%s", igw.Provider(), INTERNET_GATEWAY_TYPE, igw.Name)
}
