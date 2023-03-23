package cloudwatch

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

const LOG_GROUP_TYPE = "log_group"

var sanitizer = aws.CloudwatchLogGroupSanitizer

type (
	LogGroup struct {
		Name            string
		ConstructsRef   []core.AnnotationKey
		LogGroupName    string
		RetentionInDays int
	}
)

func NewLogGroup(appName string, logGroupName string, ref core.AnnotationKey, retention int) *LogGroup {
	return &LogGroup{
		Name:            sanitizer.Apply(fmt.Sprintf("%s-%s", appName, logGroupName)),
		ConstructsRef:   []core.AnnotationKey{ref},
		LogGroupName:    logGroupName,
		RetentionInDays: retention,
	}
}

// Provider returns name of the provider the resource is correlated to
func (image *LogGroup) Provider() string {
	return resources.AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (image *LogGroup) KlothoConstructRef() []core.AnnotationKey {
	return image.ConstructsRef
}

// ID returns the id of the cloud resource
func (image *LogGroup) Id() string {
	return fmt.Sprintf("%s_%s", LOG_GROUP_TYPE, image.Name)
}
