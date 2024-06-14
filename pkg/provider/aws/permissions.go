package aws

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/solution"
	"github.com/klothoplatform/klotho/pkg/set"
	"go.uber.org/zap"
)

// Permissions returns the permissions for the AWS provider
func DeploymentPermissionsPolicy(ctx solution.Solution) ([]byte, error) {
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

	actions := make(set.Set[string])

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
		if rt.NoIac {
			return nil
		}

		resActions := make(set.Set[string])
		resActions.Add(rt.DeploymentPermissions.Deploy...)
		resActions.Add(rt.DeploymentPermissions.TearDown...)
		resActions.Add(rt.DeploymentPermissions.Update...)
		if len(resActions) == 0 {
			zap.S().Warnf("No deployment permissions found for resource %s", resource.ID)
			return nil
		}

		actions.AddFrom(resActions)
		return nil
	})
	if err != nil {
		return nil, err
	}
	actionList := actions.ToSlice()
	sort.Strings(actionList)
	statement := map[string]any{
		"Effect":   "Allow",
		"Action":   actionList,
		"Resource": "*",
	}
	err = statementProperty.AppendProperty(policy, statement)
	if err != nil {
		return nil, err
	}

	pol, err := policy.GetProperty("Policy")

	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(pol, "", "    ")
}
