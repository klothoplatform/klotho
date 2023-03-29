package aws

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
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
	table.HashKey = "pk"
	table.RangeKey = "sk"
	if err := table.Validate(); err != nil {
		return err
	}

	a.MapResourceDirectlyToConstruct(table, kv)
	dag.AddResource(table)

	upstreamResources := result.GetUpstreamConstructs(kv)
	for _, res := range upstreamResources {
		unit, ok := res.(*core.ExecutionUnit)
		if ok {
			a.PolicyGenerator.AddAllowPolicyToUnit(unit.Id(), []string{"dynamodb:*"},
				[]core.IaCValue{
					{Resource: table, Property: core.ARN_IAC_VALUE},
					{Resource: table, Property: resources.DYNAMODB_TABLE_BACKUP_IAC_VALUE},
					{Resource: table, Property: resources.DYNAMODB_TABLE_INDEX_IAC_VALUE},
					{Resource: table, Property: resources.DYNAMODB_TABLE_EXPORT_IAC_VALUE},
					{Resource: table, Property: resources.DYNAMODB_TABLE_STREAM_IAC_VALUE},
				})
		}
	}
	return nil
}
