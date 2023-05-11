package visualizer

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

func subnetProperties(res *resources.Subnet, dag *core.ResourceGraph) map[string]any {
	return map[string]any{
		"cidr_block": res.CidrBlock,
		"public":     res.Type == resources.PublicSubnet,
	}
}
