package resources

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
	"go.uber.org/zap"
)

const (
	IAM_ROLE_TYPE       = "iam_role"
	IAM_POLICY_TYPE     = "iam_policy"
	OIDC_PROVIDER_TYPE  = "iam_oidc_provider"
	IAM_STATEMENT_ENTRY = "iam_statement_entry"
	VERSION             = "2012-10-17"
)

var roleSanitizer = aws.IamRoleSanitizer
var policySanitizer = aws.IamPolicySanitizer

var LAMBDA_ASSUMER_ROLE_POLICY = &PolicyDocument{
	Version: VERSION,
	Statement: []StatementEntry{
		{
			Action: []string{"sts:AssumeRole"},
			Principal: &Principal{
				Service: "lambda.amazonaws.com",
			},
			Effect: "Allow",
		},
	},
}

var ECS_ASSUMER_ROLE_POLICY = &PolicyDocument{
	Version: VERSION,
	Statement: []StatementEntry{
		{
			Action: []string{"sts:AssumeRole"},
			Principal: &Principal{
				Service: "ecs-tasks.amazonaws.com",
			},
			Effect: "Allow",
		},
	},
}

var EC2_ASSUMER_ROLE_POLICY = &PolicyDocument{
	Version: VERSION,
	Statement: []StatementEntry{
		{
			Action: []string{"sts:AssumeRole"},
			Principal: &Principal{
				Service: "ec2.amazonaws.com",
			},
			Effect: "Allow",
		},
	},
}

var EKS_FARGATE_ASSUME_ROLE_POLICY = &PolicyDocument{
	Version: VERSION,
	Statement: []StatementEntry{
		{
			Action: []string{"sts:AssumeRole"},
			Principal: &Principal{
				Service: "eks-fargate-pods.amazonaws.com",
			},
			Effect: "Allow",
		},
	},
}

var EKS_ASSUME_ROLE_POLICY = &PolicyDocument{
	Version: VERSION,
	Statement: []StatementEntry{
		{
			Action: []string{"sts:AssumeRole"},
			Principal: &Principal{
				Service: "eks.amazonaws.com",
			},
			Effect: "Allow",
		},
	},
}

type (
	IamRole struct {
		Name                string
		ConstructsRef       []core.AnnotationKey
		AssumeRolePolicyDoc *PolicyDocument
		ManagedPolicies     []core.IaCValue
		AwsManagedPolicies  []string
		InlinePolicies      []*IamInlinePolicy
	}

	IamPolicy struct {
		Name          string
		ConstructsRef []core.AnnotationKey
		Policy        *PolicyDocument
	}

	IamInlinePolicy struct {
		Name          string
		ConstructsRef []core.AnnotationKey
		Policy        *PolicyDocument
	}

	PolicyGenerator struct {
		unitToRole          map[string]*IamRole
		unitsInlinePolicies map[string]map[string]*IamInlinePolicy
		unitsPolicies       map[string][]*IamPolicy
	}

	PolicyDocument struct {
		Version   string
		Statement []StatementEntry
	}

	StatementEntry struct {
		Effect    string
		Action    []string
		Resource  []core.IaCValue
		Principal *Principal
		Condition *Condition
	}

	Principal struct {
		Service   string
		Federated core.IaCValue
		AWS       core.IaCValue
	}

	Condition struct {
		StringEquals map[core.IaCValue]string
		Null         map[core.IaCValue]string
	}

	OpenIdConnectProvider struct {
		Name          string
		ConstructsRef []core.AnnotationKey
		ClientIdLists []string
		Cluster       *EksCluster
		Region        *Region
	}
)

type RoleCreateParams struct {
	RoleName            string
	Refs                []core.AnnotationKey
	AssumeRolePolicyDoc *PolicyDocument
}

func (role *IamRole) Create(dag *core.ResourceGraph, params RoleCreateParams) error {
	role.Name = roleSanitizer.Apply(params.RoleName)
	role.ConstructsRef = params.Refs
	role.AssumeRolePolicyDoc = params.AssumeRolePolicyDoc

	existingRole := dag.GetResourceByVertexId(role.Id().String())
	if existingRole != nil {
		return fmt.Errorf("iam role with name %s already exists", role.Name)
	}

	dag.AddResource(role)
	return nil
}

func (lambda *IamPolicy) Create(dag *core.ResourceGraph, metadata map[string]any) (core.Resource, error) {
	panic("Not Implemented")
}

func (lambda *OpenIdConnectProvider) Create(dag *core.ResourceGraph, metadata map[string]any) (core.Resource, error) {
	panic("Not Implemented")
}

func NewPolicyGenerator() *PolicyGenerator {
	p := &PolicyGenerator{
		unitsPolicies:       make(map[string][]*IamPolicy),
		unitsInlinePolicies: make(map[string]map[string]*IamInlinePolicy),
		unitToRole:          make(map[string]*IamRole),
	}
	return p
}

func (p *PolicyGenerator) AddAllowPolicyToUnit(unitId string, policy *IamPolicy) {
	policies, found := p.unitsPolicies[unitId]
	if !found {
		p.unitsPolicies[unitId] = []*IamPolicy{policy}
	} else {
		for _, pol := range policies {
			if policy.Name == pol.Name {
				return
			}
		}
		p.unitsPolicies[unitId] = append(p.unitsPolicies[unitId], policy)
	}
}

func (p *PolicyGenerator) AddUnitRole(unitId string, role *IamRole) error {
	if _, found := p.unitToRole[unitId]; found {
		return fmt.Errorf("unit with id, %s, is already mapped to an IAM Role", unitId)
	}
	p.unitToRole[unitId] = role
	return nil
}

func (p *PolicyGenerator) AddInlinePolicyToUnit(unitId string, policy *IamInlinePolicy) {
	inlinePolicies, ok := p.unitsInlinePolicies[unitId]
	if !ok {
		p.unitsInlinePolicies[unitId] = map[string]*IamInlinePolicy{policy.Name: policy}
		return
	}
	for name := range inlinePolicies {
		if policy.Name == name {
			// TODO: handle duplicates
			zap.L().Sugar().Debugf("duplicate policy with name '%s' in unit '%s' ignored", name, unitId)
		}
	}
	p.unitsInlinePolicies[unitId][policy.Name] = policy
}

func (p *PolicyGenerator) GetUnitInlinePolicies(unitId string) []*IamInlinePolicy {
	var policies []*IamInlinePolicy
	for _, policy := range p.unitsInlinePolicies[unitId] {
		policies = append(policies, policy)
	}
	return policies
}

func (p *PolicyGenerator) GetUnitRole(unitId string) *IamRole {
	role := p.unitToRole[unitId]
	return role
}

func (p *PolicyGenerator) GetUnitPolicies(unitId string) []*IamPolicy {
	policies := p.unitsPolicies[unitId]
	return policies
}

func CreateAllowPolicyDocument(actions []string, resources []core.IaCValue) *PolicyDocument {
	return &PolicyDocument{
		Version: VERSION,
		Statement: []StatementEntry{
			{
				Effect:   "Allow",
				Action:   actions,
				Resource: resources,
			},
		},
	}
}

func NewIamRole(appName string, roleName string, ref []core.AnnotationKey, assumeRolePolicy *PolicyDocument) *IamRole {
	return &IamRole{
		Name:                roleSanitizer.Apply(fmt.Sprintf("%s-%s", appName, roleName)),
		ConstructsRef:       ref,
		AssumeRolePolicyDoc: assumeRolePolicy,
	}
}

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (role *IamRole) KlothoConstructRef() []core.AnnotationKey {
	return role.ConstructsRef
}

// Id returns the id of the cloud resource
func (role *IamRole) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     IAM_ROLE_TYPE,
		Name:     role.Name,
	}
}

// Id returns the id of the cloud resource
func (role *IamRole) AddAwsManagedPolicies(policies []string) {
	role.AwsManagedPolicies = append(role.AwsManagedPolicies, policies...)
}

func NewIamPolicy(appName string, policyName string, ref core.AnnotationKey, policy *PolicyDocument) *IamPolicy {
	return &IamPolicy{
		Name:          policySanitizer.Apply(fmt.Sprintf("%s-%s", appName, policyName)),
		ConstructsRef: []core.AnnotationKey{ref},
		Policy:        policy,
	}
}

func NewIamInlinePolicy(policyName string, ref core.AnnotationKey, policy *PolicyDocument) *IamInlinePolicy {
	return &IamInlinePolicy{
		Name:          policySanitizer.Apply(policyName),
		ConstructsRef: []core.AnnotationKey{ref},
		Policy:        policy,
	}
}

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (policy *IamPolicy) KlothoConstructRef() []core.AnnotationKey {
	return policy.ConstructsRef
}

// Id returns the id of the cloud resource
func (policy *IamPolicy) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     IAM_POLICY_TYPE,
		Name:     policy.Name,
	}
}

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (oidc *OpenIdConnectProvider) KlothoConstructRef() []core.AnnotationKey {
	return oidc.ConstructsRef
}

// Id returns the id of the cloud resource
func (oidc *OpenIdConnectProvider) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     OIDC_PROVIDER_TYPE,
		Name:     oidc.Name,
	}
}

func (s StatementEntry) Id() core.ResourceId {
	resourcesHash := sha256.New()
	for _, r := range s.Resource {
		_, _ = fmt.Fprintf(resourcesHash, "%s.%s", r.Resource.Id(), r.Property)
	}

	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     IAM_STATEMENT_ENTRY,
		Name:     fmt.Sprintf("%x/%s/%s", resourcesHash.Sum(nil), s.Effect, strings.Join(s.Action, ",")),
	}
}

func (d *PolicyDocument) Deduplicate() {
	keys := make(map[core.ResourceId]struct{})
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
