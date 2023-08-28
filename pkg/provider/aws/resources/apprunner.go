package resources

import "github.com/klothoplatform/klotho/pkg/construct"

type (
	AppRunnerService struct {
		Name                 string
		ConstructRefs        construct.BaseConstructSet `yaml:"-"`
		Image                *EcrImage
		InstanceRole         *IamRole
		EnvironmentVariables map[string]construct.IaCValue
	}
)

const (
	APP_RUNNER_SERVICE_TYPE = "app_runner_service"
)

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (lc *AppRunnerService) BaseConstructRefs() construct.BaseConstructSet {
	return lc.ConstructRefs
}

// Id returns the id of the cloud resource
func (lc *AppRunnerService) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     APP_RUNNER_SERVICE_TYPE,
		Name:     lc.Name,
	}
}

func (lc *AppRunnerService) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream:     true,
		RequiresExplicitDelete: true,
	}
}
