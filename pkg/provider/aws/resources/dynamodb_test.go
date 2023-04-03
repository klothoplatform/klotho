package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/stretchr/testify/assert"
)

func Test_NewDynamodbTable(t *testing.T) {
	assert := assert.New(t)
	construct := TestConstruct{AnnotationKey: core.AnnotationKey{
		Capability: "persist",
		ID:         "my-table",
	}}
	attributes := []DynamodbTableAttribute{
		{Name: "pk", Type: "S"},
		{Name: "sk", Type: "S"},
	}
	dynamodbTable := NewDynamodbTable(construct, "table-name", attributes)
	assert.Equal("table-name", dynamodbTable.Name)
	assert.Equal([]core.AnnotationKey{construct.Provenance()}, dynamodbTable.ConstructsRef)
	assert.Equal(PAY_PER_REQUEST, dynamodbTable.BillingMode)
	assert.Equal(attributes, dynamodbTable.Attributes)
	assert.NoError(dynamodbTable.Validate())
}

func Test_DynamodbTableProvider(t *testing.T) {
	assert := assert.New(t)
	construct := TestConstruct{AnnotationKey: core.AnnotationKey{
		Capability: "persist",
		ID:         "my-table",
	}}
	attributes := []DynamodbTableAttribute{
		{Name: "pk", Type: "S"},
		{Name: "sk", Type: "S"},
	}
	dynamodbTable := NewDynamodbTable(construct, "table-name", attributes)
	assert.Equal(dynamodbTable.Provider(), AWS_PROVIDER)
}

func Test_DynamodbTableId(t *testing.T) {
	assert := assert.New(t)
	construct := TestConstruct{AnnotationKey: core.AnnotationKey{
		Capability: "persist",
		ID:         "my-table",
	}}
	attributes := []DynamodbTableAttribute{
		{Name: "pk", Type: "S"},
		{Name: "sk", Type: "S"},
	}
	dynamodbTable := NewDynamodbTable(construct, "table-name", attributes)
	assert.Equal(dynamodbTable.Id(), "aws:dynamodb_table:table-name")
}

func Test_DynamodbTableKlothoConstructRef(t *testing.T) {
	assert := assert.New(t)
	attributes := []DynamodbTableAttribute{
		{Name: "pk", Type: "S"},
		{Name: "sk", Type: "S"},
	}
	dynamodbTable := NewDynamodbTable(nil, "table-name", attributes)
	assert.Nil(dynamodbTable.ConstructsRef)
}

type TestConstruct struct {
	AnnotationKey core.AnnotationKey
}

func (c TestConstruct) Provenance() core.AnnotationKey {
	return c.AnnotationKey
}

func (c TestConstruct) Id() string {
	return c.AnnotationKey.ID
}
