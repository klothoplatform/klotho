package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
)

func getSingleUpstreamVpc(dag *core.ResourceGraph, resource core.Resource) (vpc *Vpc, err error) {
	vpcs := core.GetAllDownstreamResourcesOfType[*Vpc](dag, resource)
	if len(vpcs) > 1 {
		return nil, fmt.Errorf("resource %s has more than one vpc downstream", resource.Id())
	} else if len(vpcs) == 1 {
		return vpcs[0], nil
	}
	return nil, nil
}
