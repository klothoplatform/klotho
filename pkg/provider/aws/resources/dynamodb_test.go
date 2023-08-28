package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/construct/coretesting"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/stretchr/testify/assert"
)

func Test_DynamodbTableCreate(t *testing.T) {
	kv := &types.Kv{Name: "test"}
	existingKey := &types.Kv{Name: "existing"}
	cases := []coretesting.CreateCase[DynamodbTableCreateParams, *DynamodbTable]{
		{
			Name: "nil dynamodb table",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:dynamodb_table:my-app-kv",
				},
			},
			Check: func(assert *assert.Assertions, table *DynamodbTable) {
				assert.Equal(table.Name, "my-app-kv")
				assert.Equal(table.ConstructRefs, construct.BaseConstructSetOf(kv))
			},
		},
		{
			Name:     "existing dynamodb table",
			Existing: &DynamodbTable{Name: "my-app-kv", ConstructRefs: construct.BaseConstructSetOf(existingKey)},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:dynamodb_table:my-app-kv",
				},
			},
			Check: func(assert *assert.Assertions, table *DynamodbTable) {
				assert.Equal(table.Name, "my-app-kv")
				assert.Equal(table.ConstructRefs, construct.BaseConstructSetOf(kv, existingKey))
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = DynamodbTableCreateParams{
				AppName: "my-app",
				Refs:    construct.BaseConstructSetOf(kv),
				Name:    "kv",
			}

			tt.Run(t)
		})
	}
}
