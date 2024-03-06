package aws

import (
	"encoding/json"
	"fmt"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/solution_context"
)

// Permissions returns the permissions for the AWS provider
func DeploymentPermissionsPolicy(ctx solution_context.SolutionContext) ([]byte, error) {
	policy := &construct.Resource{
		ID: construct.ResourceId{
			Provider: "aws",
			Type:     "iam_policy",
			Name:     "deployment_permissions",
		},
		Properties: construct.Properties{
			"Policy": map[string]any{
				"Version": "2012-10-17",
			},
		},
	}
	kb := ctx.KnowledgeBase()
	policyRt, err := kb.GetResourceTemplate(policy.ID)
	if err != nil {
		return nil, err
	}
	if policyRt == nil {
		return nil, fmt.Errorf("resource template not found for resource %s", policy.ID)
	}
	// Find the StatementProperty so we can use its methods
	statementProperty := policyRt.GetProperty("Policy.Statement")

	err = construct.WalkGraph(ctx.DataflowGraph(), func(id construct.ResourceId, resource *construct.Resource, nerr error) error {
		if nerr != nil {
			return nerr
		}
		rt, err := kb.GetResourceTemplate(resource.ID)
		if err != nil {
			return err
		}
		if rt == nil {
			return fmt.Errorf("resource template not found for resource %s", resource.ID)
		}

		statement := map[string]any{
			"Effect":   "Allow",
			"Action":   append(rt.DeploymentPermissions.Deploy, append(rt.DeploymentPermissions.TearDown, rt.DeploymentPermissions.Update...)...),
			"Resource": "*",
		}
		return statementProperty.AppendProperty(policy, statement)
	})
	if err != nil {
		return nil, err
	}
	pol, err := policy.GetProperty("Policy")

	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(pol, "", "    ")
}
