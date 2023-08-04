package resources

import "github.com/klothoplatform/klotho/pkg/core"

type (
	AppRunnerService struct {
		Name                 string
		ConstructRefs        core.BaseConstructSet `yaml:"-"`
		Image                *EcrImage
		InstanceRole         *IamRole
		EnvironmentVariables map[string]core.IaCValue
	}
)

const (
	APP_RUNNER_SERVICE_TYPE = "app_runner_service"
)

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (lc *AppRunnerService) BaseConstructRefs() core.BaseConstructSet {
	return lc.ConstructRefs
}

// Id returns the id of the cloud resource
func (lc *AppRunnerService) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     APP_RUNNER_SERVICE_TYPE,
		Name:     lc.Name,
	}
}

func (lc *AppRunnerService) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream:     true,
		RequiresExplicitDelete: true,
	}
}
