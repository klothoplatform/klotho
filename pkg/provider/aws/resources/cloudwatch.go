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
		ConstructRefs   core.BaseConstructSet `yaml:"-"`
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
	logGroup.ConstructRefs = params.Refs.Clone()

	existingLogGroup := dag.GetResource(logGroup.Id())
	if existingLogGroup != nil {
		graphLogGroup := existingLogGroup.(*LogGroup)
		graphLogGroup.ConstructRefs.AddAll(params.Refs)
	} else {
		dag.AddResource(logGroup)
	}
	return nil
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (lg *LogGroup) BaseConstructRefs() core.BaseConstructSet {
	return lg.ConstructRefs
}

// Id returns the id of the cloud resource
func (lg *LogGroup) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     LOG_GROUP_TYPE,
		Name:     lg.Name,
	}
}

func (lg *LogGroup) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream:   false,
		RequiresNoDownstream: false,
	}
}
