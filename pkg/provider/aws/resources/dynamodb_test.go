package resources

import (
	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/stretchr/testify/assert"
)

func Test_DynamodbTableCreate(t *testing.T) {
	kv := &core.Kv{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.PersistCapability}}
	existingKey := core.AnnotationKey{ID: "existing", Capability: annotation.PersistCapability}
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
				assert.ElementsMatch(table.ConstructsRef, []core.AnnotationKey{kv.AnnotationKey})
			},
		},
		{
			Name:     "existing dynamodb table",
			Existing: &DynamodbTable{Name: "my-app-kv", ConstructsRef: []core.AnnotationKey{existingKey}},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:dynamodb_table:my-app-kv",
				},
			},
			Check: func(assert *assert.Assertions, table *DynamodbTable) {
				assert.Equal(table.Name, "my-app-kv")
				assert.ElementsMatch(table.ConstructsRef, []core.AnnotationKey{kv.AnnotationKey, existingKey})
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = DynamodbTableCreateParams{
				AppName: "my-app",
				Refs:    []core.AnnotationKey{kv.AnnotationKey},
				Name:    "kv",
			}

			tt.Run(t)
		})
	}
}
