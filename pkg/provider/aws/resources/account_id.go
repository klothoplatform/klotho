package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
)

const ACCOUNT_ID_TYPE = "account_id"

type (
	AccountId struct {
		Name          string
		ConstructsRef []core.AnnotationKey
	}
)

func NewAccountId() *AccountId {
	return &AccountId{
		Name:          "accountId",
		ConstructsRef: []core.AnnotationKey{},
	}
}

// Provider returns name of the provider the resource is correlated to
func (caller *AccountId) Provider() string {
	return AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (caller *AccountId) KlothoConstructRef() []core.AnnotationKey {
	return caller.ConstructsRef
}

// ID returns the id of the cloud resource
func (caller *AccountId) Id() string {
	return fmt.Sprintf("%s:%s:%s", caller.Provider(), ACCOUNT_ID_TYPE, caller.Name)
}
