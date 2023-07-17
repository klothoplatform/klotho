package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core/coretesting"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/stretchr/testify/assert"
)

func Test_DynamodbTableCreate(t *testing.T) {
	kv := &core.Kv{Name: "test"}
	existingKey := &core.Kv{Name: "existing"}
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
				assert.Equal(table.ConstructRefs, core.BaseConstructSetOf(kv))
			},
		},
		{
			Name:     "existing dynamodb table",
			Existing: &DynamodbTable{Name: "my-app-kv", ConstructRefs: core.BaseConstructSetOf(existingKey)},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:dynamodb_table:my-app-kv",
				},
			},
			Check: func(assert *assert.Assertions, table *DynamodbTable) {
				assert.Equal(table.Name, "my-app-kv")
				assert.Equal(table.ConstructRefs, core.BaseConstructSetOf(kv, existingKey))
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = DynamodbTableCreateParams{
				AppName: "my-app",
				Refs:    core.BaseConstructSetOf(kv),
				Name:    "kv",
			}

			tt.Run(t)
		})
	}
}
