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
		ConstructsRef   core.BaseConstructSet `yaml:"-"`
		LogGroupName    string
		RetentionInDays int
	}
)

type CloudwatchLogGroupCreateParams struct {
	AppName string
	Refs    core.BaseConstructSet
	Name    string
}

func (logGroup *LogGroup) Create(dag *core.ResourceGraph, params CloudwatchLogGroupCreateParams) error {
	logGroup.Name = logGroupSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	logGroup.ConstructsRef = params.Refs.Clone()

	existingLogGroup := dag.GetResource(logGroup.Id())
	if existingLogGroup != nil {
		graphLogGroup := existingLogGroup.(*LogGroup)
		graphLogGroup.ConstructsRef.AddAll(params.Refs)
	} else {
		dag.AddResource(logGroup)
	}
	return nil
}

// BaseConstructsRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (lg *LogGroup) BaseConstructsRef() core.BaseConstructSet {
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

func (lg *LogGroup) DeleteCriteria() core.DeleteCriteria {
	return core.DeleteCriteria{
		RequiresNoUpstream:   false,
		RequiresNoDownstream: false,
	}
}
