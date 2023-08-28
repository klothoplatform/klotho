package visualizer

import (
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

func TypeFor(res construct.Resource, dag *construct.ResourceGraph) string {
	resType := res.Id().Type
	// Important: if you update this switch, also update all_types_test.go's typeNamesForResource
	switch res := res.(type) {
	case *resources.Subnet:
		resType = "subnet" // not "vpc_subnet"
	case *resources.VpcEndpoint:
		switch res.VpcEndpointType {
		case "Interface":
			resType = "vpc_endpoint_interface"
		case "Gateway":
			resType = "vpc_endpoint_gateway"
		}
	}
	return resType
}
