package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

const DYNAMODB_TABLE_TYPE = "dynamodb_table"

type (
	DynamodbTable struct {
		Name          string
		ConstructRefs construct.BaseConstructSet `yaml:"-"`
		Attributes    []DynamodbTableAttribute
		BillingMode   string
		HashKey       string
		RangeKey      string
	}

	DynamodbTableAttribute struct {
		Name string
		Type string
	}
)

const (
	DYNAMODB_TABLE_STREAM_IAC_VALUE = "dynamodb_table__stream"
	DYNAMODB_TABLE_BACKUP_IAC_VALUE = "dynamodb_table__backup"
	DYNAMODB_TABLE_EXPORT_IAC_VALUE = "dynamodb_table__export"
	DYNAMODB_TABLE_INDEX_IAC_VALUE  = "dynamodb_table__index"
)

type DynamodbTableCreateParams struct {
	AppName string
	Refs    construct.BaseConstructSet
	Name    string
}

func (table *DynamodbTable) Create(dag *construct.ResourceGraph, params DynamodbTableCreateParams) error {
	table.Name = aws.DynamoDBTableSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	table.ConstructRefs = params.Refs.Clone()
	if existingTable, ok := construct.GetResource[*DynamodbTable](dag, table.Id()); ok {
		existingTable.ConstructRefs.AddAll(params.Refs)
	}
	dag.AddResource(table)
	return nil
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (table *DynamodbTable) BaseConstructRefs() construct.BaseConstructSet {
	return table.ConstructRefs
}

// Id returns the id of the cloud resource
func (table *DynamodbTable) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     DYNAMODB_TABLE_TYPE,
		Name:     table.Name,
	}
}

func (table *DynamodbTable) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream:     true,
		RequiresNoDownstream:   true,
		RequiresExplicitDelete: true,
	}
}
