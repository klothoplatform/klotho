package aws

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

const API_GATEWAY_EXECUTION_CHILD_RESOURCES_IAC_VALUE = "child_resources"

// expandOrm takes in a single orm construct and expands the generic construct into a set of resource's based on the units configuration.
func (a *AWS) expandExpose(dag *core.ResourceGraph, expose *core.Gateway) error {
	switch a.Config.GetExpose(expose.ID).Type {
	case ApiGateway:
		stage, err := core.CreateResource[*resources.ApiStage](dag, resources.ApiStageCreateParams{
			AppName: a.Config.AppName,
			Refs:    core.AnnotationKeySetOf(expose.AnnotationKey),
			Name:    expose.ID,
		})
		if err != nil {
			return err
		}
		err = a.MapResourceToConstruct(stage.RestApi, expose)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported expose type %s", a.Config.GetExpose(expose.ID).Type)
	}
	return nil
}
