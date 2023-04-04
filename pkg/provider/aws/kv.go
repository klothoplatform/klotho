package aws

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"go.uber.org/zap"
)

func (a *AWS) GenerateKvResources(kv *core.Kv, result *core.ConstructGraph, dag *core.ResourceGraph) error {
	table := resources.NewDynamodbTable(
		kv,
		fmt.Sprintf("KV_%s", a.Config.AppName),
		[]resources.DynamodbTableAttribute{
			{Name: "pk", Type: "S"},
			{Name: "sk", Type: "S"},
		},
	)

	if existingTable := dag.GetResource(table.Id()); existingTable == nil {
		table.HashKey = "pk"
		table.RangeKey = "sk"
		if err := table.Validate(); err != nil {
			return err
		}
		dag.AddResource(table)
	} else {
		table = existingTable.(*resources.DynamodbTable)
		table.ConstructsRef = append(table.ConstructsRef, kv.AnnotationKey)
		zap.L().Sugar().Debugf("skipping resource generation for [construct -> resource] relationship, [%s -> %s]: target resource already exists in the application's resource graph.", kv.ID, table.Id())
	}

	a.MapResourceDirectlyToConstruct(table, kv)

	upstreamConstructs := result.GetUpstreamConstructs(kv)
	for _, res := range upstreamConstructs {
		unit, ok := res.(*core.ExecutionUnit)
		if ok {
			actions := []string{"dynamodb:*"}
			policyResources := []core.IaCValue{
				{Resource: table, Property: core.ARN_IAC_VALUE},
				{Resource: table, Property: resources.DYNAMODB_TABLE_BACKUP_IAC_VALUE},
				{Resource: table, Property: resources.DYNAMODB_TABLE_INDEX_IAC_VALUE},
				{Resource: table, Property: resources.DYNAMODB_TABLE_EXPORT_IAC_VALUE},
				{Resource: table, Property: resources.DYNAMODB_TABLE_STREAM_IAC_VALUE},
			}
			policyDoc := resources.CreateAllowPolicyDocument(actions, policyResources)
			policy := resources.NewIamPolicy(a.Config.AppName, kv.Id(), kv.Provenance(), policyDoc)
			dag.AddResource(policy)
			dag.AddDependency2(policy, table)
			a.PolicyGenerator.AddAllowPolicyToUnit(unit.Id(), policy)
		}
	}
	return nil
}
