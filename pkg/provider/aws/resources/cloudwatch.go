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

func (lambda *LogGroup) Create(dag *core.ResourceGraph, metadata map[string]any) (core.Resource, error) {
	panic("Not Implemented")
}

func NewLogGroup(appName string, logGroupName string, ref core.AnnotationKey, retention int) *LogGroup {
	return &LogGroup{
		Name:            logGroupSanitizer.Apply(fmt.Sprintf("%s-%s", appName, logGroupName)),
		ConstructsRef:   []core.AnnotationKey{ref},
		LogGroupName:    logGroupSanitizer.Apply(logGroupName),
		RetentionInDays: retention,
	}
}

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (lg *LogGroup) KlothoConstructRef() []core.AnnotationKey {
	return lg.ConstructsRef
}

// Id returns the id of the cloud resource
func (lg *LogGroup) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     LOG_GROUP_TYPE,
		Name:     lg.Name,
	}
}
