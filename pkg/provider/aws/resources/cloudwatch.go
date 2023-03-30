package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

const LOG_GROUP_TYPE = "log_group"

var logGroupSanitizer = aws.CloudwatchLogGroupSanitizer

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
		Name:            logGroupSanitizer.Apply(fmt.Sprintf("%s-%s", appName, logGroupName)),
		ConstructsRef:   []core.AnnotationKey{ref},
		LogGroupName:    logGroupName,
		RetentionInDays: retention,
	}
}

// Provider returns name of the provider the resource is correlated to
func (lg *LogGroup) Provider() string {
	return AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (lg *LogGroup) KlothoConstructRef() []core.AnnotationKey {
	return lg.ConstructsRef
}

// ID returns the id of the cloud resource
func (lg *LogGroup) Id() string {
	return fmt.Sprintf("%s:%s:%s", lg.Provider(), LOG_GROUP_TYPE, lg.Name)
}
