package iam

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

const IAM_ROLE_TYPE = "iam_role"

var sanitizer = aws.IamRoleSanitizer

type (
	IamRole struct {
		Name                string
		ConstructsRef       []core.AnnotationKey
		AssumeRolePolicyDoc string
		ManagedPolicyArns   []string
	}
)

func NewIamRole(appName string, roleName string, ref core.AnnotationKey, assumeRolePolicy string) *IamRole {
	return &IamRole{
		Name:                sanitizer.Apply(fmt.Sprintf("%s-%s", appName, roleName)),
		ConstructsRef:       []core.AnnotationKey{ref},
		AssumeRolePolicyDoc: assumeRolePolicy,
	}
}

// Provider returns name of the provider the resource is correlated to
func (role *IamRole) Provider() string {
	return resources.AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (role *IamRole) KlothoConstructRef() []core.AnnotationKey {
	return role.ConstructsRef
}

// ID returns the id of the cloud resource
func (role *IamRole) Id() string {
	return fmt.Sprintf("%s:%s:%s", role.Provider(), IAM_ROLE_TYPE, role.Name)
}

const LAMBDA_ASSUMER_ROLE_POLICY = `{
	Version: '2012-10-17',
	Statement: [
		{
			Action: 'sts:AssumeRole',
			Principal: {
				Service: 'lambda.amazonaws.com',
			},
			Effect: 'Allow',
			Sid: '',
		},
	],
}`

const ECS_ASSUMER_ROLE_POLICY = `{
	Version: '2012-10-17',
	Statement: [
		{
			Action: 'sts:AssumeRole',
			Principal: {
				Service: 'ecs-tasks.amazonaws.com',
			},
			Effect: 'Allow',
			Sid: '',
		},
	],
}`

const EC2_ASSUMER_ROLE_POLICY = `{
	Version: '2012-10-17',
	Statement: [
		{
			Action: 'sts:AssumeRole',
			Principal: {
				Service: 'ec2.amazonaws.com',
			},
			Effect: 'Allow',
			Sid: '',
		},
	],
}`

const EKS_FARGATE_ASSUME_ROLE_POLICY = `{
	Version: '2012-10-17',
	Statement: [
		{
			Action: 'sts:AssumeRole',
			Principal: {
				Service: 'eks-fargate-pods.amazonaws.com',
			},
			Effect: 'Allow',
			Sid: '',
		},
	],
}`
