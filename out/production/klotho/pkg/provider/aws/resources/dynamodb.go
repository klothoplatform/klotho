package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
	"github.com/pkg/errors"
)

const DYNAMODB_TABLE_TYPE = "dynamodb_table"

type (
	DynamodbTable struct {
		Name          string
		ConstructsRef core.AnnotationKeySet
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
	PAY_PER_REQUEST string = "PAY_PER_REQUEST"
	PROVISIONED     string = "PROVISIONED"

	DYNAMODB_TABLE_STREAM_IAC_VALUE = "dynamodb_table__stream"
	DYNAMODB_TABLE_BACKUP_IAC_VALUE = "dynamodb_table__backup"
	DYNAMODB_TABLE_EXPORT_IAC_VALUE = "dynamodb_table__export"
	DYNAMODB_TABLE_INDEX_IAC_VALUE  = "dynamodb_table__index"
)

func (table *DynamodbTable) Validate() error {
	var merr multierr.Error
	if table.BillingMode != PAY_PER_REQUEST && table.BillingMode != PROVISIONED {
		merr.Append(fmt.Errorf("invalid billing mode: '%s'. billing mode must be one of: ['%s', '%s']", table.BillingMode, PROVISIONED, PAY_PER_REQUEST))
	}
	attrMap, err := table.AttributeMap()
	merr.Append(err)
	if err == nil {
		if _, ok := attrMap[table.HashKey]; table.HashKey != "" && !ok {
			merr.Append(fmt.Errorf("hashKey '%s' not present in attributes", table.HashKey))
		}

		if _, ok := attrMap[table.RangeKey]; table.RangeKey != "" && !ok {
			merr.Append(fmt.Errorf("rangeKey '%s' not present in attributes", table.RangeKey))
		}
	}
	return merr.ErrOrNil()
}

func (table *DynamodbTable) AttributeMap() (map[string]DynamodbTableAttribute, error) {
	var merr multierr.Error
	attrs := make(map[string]DynamodbTableAttribute)
	for _, attr := range table.Attributes {
		if attr.Name == "" {
			merr.Append(errors.New("invalid table attribute: missing name"))
			continue
		}
		if attr.Type == "" {
			merr.Append(errors.New("invalid table attribute: missing type"))
			continue
		}
		if _, ok := attrs[attr.Name]; ok {
			merr.Append(fmt.Errorf("duplicate table attribute: '%s'", attr.Name))
			continue
		} else {
			attrs[attr.Name] = attr
		}
	}
	return attrs, merr.ErrOrNil()
}

type DynamodbTableCreateParams struct {
	AppName string
	Refs    core.AnnotationKeySet
	Name    string
}

func (table *DynamodbTable) Create(dag *core.ResourceGraph, params DynamodbTableCreateParams) error {
	table.Name = aws.DynamoDBTableSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	table.ConstructsRef = params.Refs
	if existingTable, ok := core.GetResource[*DynamodbTable](dag, table.Id()); ok {
		existingTable.ConstructsRef.AddAll(params.Refs)
	}
	dag.AddResource(table)
	return nil
}

type DynamodbTableConfigureParams struct {
	Attributes  []DynamodbTableAttribute
	BillingMode string
	HashKey     string
	RangeKey    string
}

func (table *DynamodbTable) Configure(params DynamodbTableConfigureParams) error {
	table.Attributes = params.Attributes
	table.BillingMode = params.BillingMode
	table.HashKey = params.HashKey
	table.RangeKey = params.RangeKey
	return nil
}

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (table *DynamodbTable) KlothoConstructRef() core.AnnotationKeySet {
	return table.ConstructsRef
}

// Id returns the id of the cloud resource
func (table *DynamodbTable) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     DYNAMODB_TABLE_TYPE,
		Name:     table.Name,
	}
}
