package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_KmsKeyCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test"}
	eu2 := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu2)
	cases := []coretesting.CreateCase[KmsKeyCreateParams, *KmsKey]{
		{
			Name: "nil kms key",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:kms_key:my-app-key",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, record *KmsKey) {
				assert.Equal(record.Name, "my-app-key")
				assert.Equal(record.ConstructRefs, core.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "existing kms key",
			Existing: &KmsKey{Name: "my-app-key", ConstructRefs: initialRefs},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:kms_key:my-app-key",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, record *KmsKey) {
				assert.Equal(record.Name, "my-app-key")
				initialRefs.Add(eu)
				assert.Equal(record.ConstructRefs, initialRefs)
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = KmsKeyCreateParams{
				Refs:    core.BaseConstructSetOf(eu),
				AppName: "my-app",
				Name:    "key",
			}
			tt.Run(t)
		})
	}
}

func Test_KmsAliasCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test"}
	eu2 := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu2)
	cases := []coretesting.CreateCase[KmsAliasCreateParams, *KmsAlias]{
		{
			Name: "nil kms key",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:kms_key:my-app-key",
					"aws:kms_alias:my-app-key-alias",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:kms_alias:my-app-key-alias", Destination: "aws:kms_key:my-app-key"},
				},
			},
			Check: func(assert *assert.Assertions, record *KmsAlias) {
				assert.Equal(record.Name, "my-app-key-alias")
				assert.NotNil(record.TargetKey)
				assert.Equal(record.AliasName, "alias/alias")
				assert.Equal(record.ConstructRefs, core.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "existing kms key",
			Existing: &KmsAlias{Name: "my-app-key-alias", ConstructRefs: initialRefs},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:kms_alias:my-app-key-alias",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, record *KmsAlias) {
				assert.Equal(record.Name, "my-app-key-alias")
				initialRefs.Add(eu)
				assert.Equal(record.ConstructRefs, initialRefs)
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = KmsAliasCreateParams{
				Key:  &KmsKey{Name: "my-app-key", ConstructRefs: core.BaseConstructSetOf(eu)},
				Name: "alias",
			}
			tt.Run(t)
		})
	}
}

func Test_KmsReplicaKeyCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test"}
	eu2 := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu2)
	cases := []coretesting.CreateCase[KmsReplicaKeyCreateParams, *KmsReplicaKey]{
		{
			Name: "nil kms key replica",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:kms_key:my-app-key",
					"aws:kms_replica_key:my-app-key-replica",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:kms_replica_key:my-app-key-replica", Destination: "aws:kms_key:my-app-key"},
				},
			},
			Check: func(assert *assert.Assertions, record *KmsReplicaKey) {
				assert.Equal(record.Name, "my-app-key-replica")
				assert.NotNil(record.PrimaryKey)
				assert.Equal(record.ConstructRefs, core.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "existing kms key replica",
			Existing: &KmsReplicaKey{Name: "my-app-key-replica", ConstructRefs: initialRefs},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:kms_replica_key:my-app-key-replica",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, record *KmsReplicaKey) {
				assert.Equal(record.Name, "my-app-key-replica")
				initialRefs.Add(eu)
				assert.Equal(record.ConstructRefs, initialRefs)
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = KmsReplicaKeyCreateParams{
				Key:  &KmsKey{Name: "my-app-key", ConstructRefs: core.BaseConstructSetOf(eu)},
				Name: "replica",
			}
			tt.Run(t)
		})
	}
}
