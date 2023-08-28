package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

const (
	KMS_KEY_TYPE         = "kms_key"
	KMS_ALIAS_TYPE       = "kms_alias"
	KMS_REPLICA_KEY_TYPE = "kms_replica_key"
)

type (
	KmsKey struct {
		Name                string
		ConstructRefs       construct.BaseConstructSet `yaml:"-"`
		Description         string
		Enabled             bool
		EnableKeyRotation   bool
		KeyPolicy           *PolicyDocument
		KeySpec             string
		KeyUsage            string
		MultiRegion         bool
		PendingWindowInDays int
	}

	KmsAlias struct {
		Name          string
		ConstructRefs construct.BaseConstructSet `yaml:"-"`
		AliasName     string
		TargetKey     *KmsKey
	}

	KmsReplicaKey struct {
		Name                string
		ConstructRefs       construct.BaseConstructSet `yaml:"-"`
		Description         string
		Enabled             bool
		KeyPolicy           *PolicyDocument
		PendingWindowInDays int
		PrimaryKey          *KmsKey
	}
)

type KmsKeyCreateParams struct {
	AppName string
	Refs    construct.BaseConstructSet
	Name    string
}

func (key *KmsKey) Create(dag *construct.ResourceGraph, params KmsKeyCreateParams) error {

	name := aws.KmsKeySanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	key.Name = name
	key.ConstructRefs = params.Refs

	existingKey, found := construct.GetResource[*KmsKey](dag, key.Id())
	if found {
		existingKey.ConstructRefs.AddAll(params.Refs)
		return nil
	}
	dag.AddResource(key)
	return nil
}

type KmsAliasCreateParams struct {
	Key  *KmsKey
	Name string
}

func (alias *KmsAlias) Create(dag *construct.ResourceGraph, params KmsAliasCreateParams) error {

	name := aws.KmsKeySanitizer.Apply(fmt.Sprintf("%s-%s", params.Key.Name, params.Name))
	alias.Name = name
	alias.ConstructRefs = params.Key.ConstructRefs.Clone()
	alias.TargetKey = params.Key
	alias.AliasName = aws.KmsKeySanitizer.Apply(fmt.Sprintf("alias/%s", params.Name))
	existingKey, found := construct.GetResource[*KmsAlias](dag, alias.Id())
	if found {
		existingKey.ConstructRefs.AddAll(params.Key.ConstructRefs)
		return nil
	}
	dag.AddDependenciesReflect(alias)
	return nil
}

type KmsReplicaKeyCreateParams struct {
	Key  *KmsKey
	Name string
}

func (key *KmsReplicaKey) Create(dag *construct.ResourceGraph, params KmsReplicaKeyCreateParams) error {

	name := aws.KmsKeySanitizer.Apply(fmt.Sprintf("%s-%s", params.Key.Name, params.Name))
	key.Name = name
	key.ConstructRefs = params.Key.ConstructRefs.Clone()
	key.PrimaryKey = params.Key
	existingKey, found := construct.GetResource[*KmsReplicaKey](dag, key.Id())
	if found {
		existingKey.ConstructRefs.AddAll(params.Key.ConstructRefs)
		return nil
	}
	dag.AddDependenciesReflect(key)
	return nil
}

type KmsReplicaKeyConfigureParams struct {
}

func (key *KmsReplicaKey) Configure(params KmsReplicaKeyConfigureParams) error {
	key.Enabled = true
	key.PendingWindowInDays = 7
	return nil
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (key *KmsKey) BaseConstructRefs() construct.BaseConstructSet {
	return key.ConstructRefs
}

// Id returns the id of the cloud resource
func (key *KmsKey) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     KMS_KEY_TYPE,
		Name:     key.Name,
	}
}

func (key *KmsKey) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream: true,
	}
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (alias *KmsAlias) BaseConstructRefs() construct.BaseConstructSet {
	return alias.ConstructRefs
}

// Id returns the id of the cloud resource
func (alias *KmsAlias) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     KMS_ALIAS_TYPE,
		Name:     alias.Name,
	}
}

func (alias *KmsAlias) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream: true,
	}
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (replica *KmsReplicaKey) BaseConstructRefs() construct.BaseConstructSet {
	return replica.ConstructRefs
}

// Id returns the id of the cloud resource
func (replica *KmsReplicaKey) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     KMS_REPLICA_KEY_TYPE,
		Name:     replica.Name,
	}
}

func (replica *KmsReplicaKey) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream: true,
	}
}
