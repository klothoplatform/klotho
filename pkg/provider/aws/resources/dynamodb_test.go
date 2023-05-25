package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"

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
				assert.Equal(table.ConstructsRef, core.AnnotationKeySetOf(kv.AnnotationKey))
			},
		},
		{
			Name:     "existing dynamodb table",
			Existing: []core.Resource{&DynamodbTable{Name: "my-app-kv", ConstructsRef: core.AnnotationKeySetOf(existingKey)}},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:dynamodb_table:my-app-kv",
				},
			},
			Check: func(assert *assert.Assertions, table *DynamodbTable) {
				assert.Equal(table.Name, "my-app-kv")
				assert.Equal(table.ConstructsRef, core.AnnotationKeySetOf(kv.AnnotationKey, existingKey))
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = DynamodbTableCreateParams{
				AppName: "my-app",
				Refs:    core.AnnotationKeySetOf(kv.AnnotationKey),
				Name:    "kv",
			}

			tt.Run(t)
		})
	}
}
