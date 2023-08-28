package visualizer

import (
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

func subnetProperties(res *resources.Subnet, dag *construct.ResourceGraph) map[string]any {
	return map[string]any{
		"cidr_block": res.CidrBlock,
		"public":     res.Type == resources.PublicSubnet,
	}
}

func rdsInstanceProperties(res *resources.RdsInstance, dag *construct.ResourceGraph) map[string]any {
	return map[string]any{
		"engine": res.Engine,
	}
}
