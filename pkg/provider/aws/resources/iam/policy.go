package iam

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

const IAM_POLICY_TYPE = "iam_policy"

var policySanitizer = aws.IamPolicySanitizer

type (
	IamPolicy struct {
		Name          string
		ConstructsRef []core.AnnotationKey
		Policy        *PolicyDocument `render:"document"`
	}
)

func NewIamPolicy(appName string, policyName string, ref core.AnnotationKey, policy *PolicyDocument) *IamPolicy {
	return &IamPolicy{
		Name:          policySanitizer.Apply(fmt.Sprintf("%s-%s", appName, policyName)),
		ConstructsRef: []core.AnnotationKey{ref},
		Policy:        policy,
	}
}

// Provider returns name of the provider the resource is correlated to
func (policy *IamPolicy) Provider() string {
	return resources.AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (policy *IamPolicy) KlothoConstructRef() []core.AnnotationKey {
	return policy.ConstructsRef
}

// ID returns the id of the cloud resource
func (policy *IamPolicy) Id() string {
	return fmt.Sprintf("%s:%s:%s", policy.Provider(), IAM_POLICY_TYPE, policy.Name)
}
