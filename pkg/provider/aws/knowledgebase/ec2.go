package knowledgebase

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/construct"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

var Ec2KB = knowledgebase.Build(
	knowledgebase.EdgeBuilder[*resources.Ec2Instance, *resources.EfsMountTarget]{
		// Even with this edge configured, a user still needs to mount the EFS volume manually. See: https://docs.aws.amazon.com/efs/latest/ug/wt1-test.html, https://docs.aws.amazon.com/efs/latest/ug/mounting-fs-mount-helper-ec2-linux.html
		Configure: func(instance *resources.Ec2Instance, mountTarget *resources.EfsMountTarget, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
			efsVpc, err := construct.GetSingleDownstreamResourceOfType[*resources.Vpc](dag, mountTarget)
			if err != nil {
				return err
			}
			serviceVpc, _ := construct.GetSingleDownstreamResourceOfType[*resources.Vpc](dag, instance)

			if serviceVpc != nil && efsVpc != nil && serviceVpc != efsVpc {
				return fmt.Errorf("instance %s and efs access point %s must be in the same vpc", instance.Id(), mountTarget.Id())
			}

			if serviceVpc == nil {
				dag.AddDependencyWithData(instance, efsVpc, data)
			}

			return nil
		},
	},
)
