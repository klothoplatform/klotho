package resources

import (
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

const (
	IAM_ROLE_TYPE              = "iam_role"
	IAM_POLICY_TYPE            = "iam_policy"
	LAMBDA_ASSUMER_ROLE_POLICY = `{
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

	ECS_ASSUMER_ROLE_POLICY = `{
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

	EC2_ASSUMER_ROLE_POLICY = `{
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

	EKS_FARGATE_ASSUME_ROLE_POLICY = `{
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
	VERSION = "2012-10-17"
)

var roleSanitizer = aws.IamRoleSanitizer
var policySanitizer = aws.IamPolicySanitizer

type (
	IamRole struct {
		Name                string
		ConstructsRef       []core.AnnotationKey
		AssumeRolePolicyDoc string
		ManagedPolicies     []core.IaCValue
		AwsManagedPolicies  []string
		InlinePolicy        *PolicyDocument `render:"document"`
	}

	IamPolicy struct {
		Name          string
		ConstructsRef []core.AnnotationKey
		Policy        *PolicyDocument `render:"document"`
	}

	PolicyGenerator struct {
		unitToRole    map[string]*IamRole
		unitsPolicies map[string]*PolicyDocument
	}

	PolicyDocument struct {
		Version   string
		Statement []StatementEntry `render:"document"`
	}

	StatementEntry struct {
		Effect   string
		Action   []string
		Resource []core.IaCValue
	}
)

func NewPolicyGenerator() *PolicyGenerator {
	p := &PolicyGenerator{
		unitsPolicies: make(map[string]*PolicyDocument),
		unitToRole:    make(map[string]*IamRole),
	}
	return p
}

func (p *PolicyGenerator) AddAllowPolicyToUnit(unitId string, actions []string, resources []core.IaCValue) {
	policyDoc, found := p.unitsPolicies[unitId]
	if !found {
		p.unitsPolicies[unitId] = &PolicyDocument{
			Version: VERSION,
			Statement: []StatementEntry{
				{
					Effect:   "Allow",
					Action:   actions,
					Resource: resources,
				},
			},
		}
	} else {
		policyDoc.Statement = append(policyDoc.Statement, StatementEntry{
			Effect:   "Allow",
			Action:   actions,
			Resource: resources,
		})
		policyDoc.Deduplicate()
	}
}

func (p *PolicyGenerator) AddUnitRole(unitId string, role *IamRole) error {
	if _, found := p.unitToRole[unitId]; found {
		return fmt.Errorf("unit with id, %s, is already mapped to an IAM Role", unitId)
	}
	p.unitToRole[unitId] = role
	return nil
}

func (p *PolicyGenerator) GetUnitRole(unitId string) *IamRole {
	role := p.unitToRole[unitId]
	return role
}

func (p *PolicyGenerator) GetUnitPolicies(unitId string) *PolicyDocument {
	policyDoc := p.unitsPolicies[unitId]
	return policyDoc
}

func NewIamRole(appName string, roleName string, ref core.AnnotationKey, assumeRolePolicy string) *IamRole {
	return &IamRole{
		Name:                roleSanitizer.Apply(fmt.Sprintf("%s-%s", appName, roleName)),
		ConstructsRef:       []core.AnnotationKey{ref},
		AssumeRolePolicyDoc: assumeRolePolicy,
	}
}

// Provider returns name of the provider the resource is correlated to
func (role *IamRole) Provider() string {
	return AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (role *IamRole) KlothoConstructRef() []core.AnnotationKey {
	return role.ConstructsRef
}

// ID returns the id of the cloud resource
func (role *IamRole) Id() string {
	return fmt.Sprintf("%s:%s:%s", role.Provider(), IAM_ROLE_TYPE, role.Name)
}

func NewIamPolicy(appName string, policyName string, ref core.AnnotationKey, policy *PolicyDocument) *IamPolicy {
	return &IamPolicy{
		Name:          policySanitizer.Apply(fmt.Sprintf("%s-%s", appName, policyName)),
		ConstructsRef: []core.AnnotationKey{ref},
		Policy:        policy,
	}
}

// Provider returns name of the provider the resource is correlated to
func (policy *IamPolicy) Provider() string {
	return AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (policy *IamPolicy) KlothoConstructRef() []core.AnnotationKey {
	return policy.ConstructsRef
}

// ID returns the id of the cloud resource
func (policy *IamPolicy) Id() string {
	return fmt.Sprintf("%s:%s:%s", policy.Provider(), IAM_POLICY_TYPE, policy.Name)
}

func (s StatementEntry) Id() string {
	var id strings.Builder
	for _, r := range s.Resource {
		id.WriteString(r.Resource.Id())
		id.WriteRune(':')
		id.WriteString(r.Property)
	}
	id.WriteString("::")
	id.WriteString(s.Effect)
	id.WriteString("::")
	id.WriteString(strings.Join(s.Action, ","))

	return id.String()
}

func (d *PolicyDocument) Deduplicate() {
	keys := make(map[string]struct{})
	var unique []StatementEntry
	for _, stmt := range d.Statement {
		id := stmt.Id()
		if _, ok := keys[id]; !ok {
			keys[id] = struct{}{}
			unique = append(unique, stmt)
		}
	}
	d.Statement = unique
}
