package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
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
		ConstructsRef       core.BaseConstructSet `yaml:"-"`
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
		ConstructsRef core.BaseConstructSet `yaml:"-"`
		AliasName     string
		TargetKey     *KmsKey
	}

	KmsReplicaKey struct {
		Name                string
		ConstructsRef       core.BaseConstructSet `yaml:"-"`
		Description         string
		Enabled             bool
		KeyPolicy           *PolicyDocument
		PendingWindowInDays int
		PrimaryKey          *KmsKey
	}
)

type KmsKeyCreateParams struct {
	AppName string
	Refs    core.BaseConstructSet
	Name    string
}

func (key *KmsKey) Create(dag *core.ResourceGraph, params KmsKeyCreateParams) error {

	name := aws.KmsKeySanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	key.Name = name
	key.ConstructsRef = params.Refs

	existingKey, found := core.GetResource[*KmsKey](dag, key.Id())
	if found {
		existingKey.ConstructsRef.AddAll(params.Refs)
		return nil
	}
	dag.AddResource(key)
	return nil
}

type KmsKeyConfigureParams struct {
}

func (key *KmsKey) Configure(params KmsKeyConfigureParams) error {
	key.EnableKeyRotation = true
	key.Enabled = true
	key.MultiRegion = false
	key.KeySpec = "SYMMETRIC_DEFAULT"
	key.KeyUsage = "ENCRYPT_DECRYPT"
	key.PendingWindowInDays = 7
	return nil
}

type KmsAliasCreateParams struct {
	Key  *KmsKey
	Name string
}

func (alias *KmsAlias) Create(dag *core.ResourceGraph, params KmsAliasCreateParams) error {

	name := aws.KmsKeySanitizer.Apply(fmt.Sprintf("%s-%s", params.Key.Name, params.Name))
	alias.Name = name
	alias.ConstructsRef = params.Key.ConstructsRef.Clone()
	alias.TargetKey = params.Key
	alias.AliasName = aws.KmsKeySanitizer.Apply(fmt.Sprintf("alias/%s", params.Name))
	existingKey, found := core.GetResource[*KmsAlias](dag, alias.Id())
	if found {
		existingKey.ConstructsRef.AddAll(params.Key.ConstructsRef)
		return nil
	}
	dag.AddDependenciesReflect(alias)
	return nil
}

type KmsReplicaKeyCreateParams struct {
	Key  *KmsKey
	Name string
}

func (key *KmsReplicaKey) Create(dag *core.ResourceGraph, params KmsReplicaKeyCreateParams) error {

	name := aws.KmsKeySanitizer.Apply(fmt.Sprintf("%s-%s", params.Key.Name, params.Name))
	key.Name = name
	key.ConstructsRef = params.Key.ConstructsRef.Clone()
	key.PrimaryKey = params.Key
	existingKey, found := core.GetResource[*KmsReplicaKey](dag, key.Id())
	if found {
		existingKey.ConstructsRef.AddAll(params.Key.ConstructsRef)
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

// BaseConstructsRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (key *KmsKey) BaseConstructsRef() core.BaseConstructSet {
	return key.ConstructsRef
}

// Id returns the id of the cloud resource
func (key *KmsKey) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     KMS_KEY_TYPE,
		Name:     key.Name,
	}
}

func (key *KmsKey) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}

// BaseConstructsRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (alias *KmsAlias) BaseConstructsRef() core.BaseConstructSet {
	return alias.ConstructsRef
}

// Id returns the id of the cloud resource
func (alias *KmsAlias) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     KMS_ALIAS_TYPE,
		Name:     alias.Name,
	}
}

func (alias *KmsAlias) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}

// BaseConstructsRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (replica *KmsReplicaKey) BaseConstructsRef() core.BaseConstructSet {
	return replica.ConstructsRef
}

// Id returns the id of the cloud resource
func (replica *KmsReplicaKey) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     KMS_REPLICA_KEY_TYPE,
		Name:     replica.Name,
	}
}

func (replica *KmsReplicaKey) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}
