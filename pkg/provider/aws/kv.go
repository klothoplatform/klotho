package aws

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

func (a *AWS) expandKv(dag *core.ResourceGraph, kv *core.Kv) error {
	table, err := core.CreateResource[*resources.DynamodbTable](dag, resources.DynamodbTableCreateParams{
		AppName: a.Config.AppName,
		Refs:    []core.AnnotationKey{kv.AnnotationKey},
		Name:    "kv",
	})
	if err != nil {
		return err
	}

	err = a.MapResourceToConstruct(table, kv)
	if err != nil {
		return err
	}
	return nil
}

func (a *AWS) getKvConfiguration() resources.DynamodbTableConfigureParams {
	return resources.DynamodbTableConfigureParams{
		Attributes: []resources.DynamodbTableAttribute{
			{Name: "pk", Type: "S"},
			{Name: "sk", Type: "S"},
		},
		BillingMode: resources.PAY_PER_REQUEST,
		HashKey:     "pk",
		RangeKey:    "sk",
	}
}
