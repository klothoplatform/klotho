package resources

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

const (
	IAM_ROLE_TYPE                   = "iam_role"
	IAM_POLICY_TYPE                 = "iam_policy"
	OIDC_PROVIDER_TYPE              = "iam_oidc_provider"
	IAM_STATEMENT_ENTRY             = "iam_statement_entry"
	IAM_ROLE_POLICY_ATTACHMENT_TYPE = "role_policy_attachment"
	VERSION                         = "2012-10-17"
	INSTANCE_PROFILE_TYPE           = "iam_instance_profile"
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
		ConstructsRef       core.AnnotationKeySet
		AssumeRolePolicyDoc *PolicyDocument
		ManagedPolicies     []core.IaCValue
		AwsManagedPolicies  []string
		InlinePolicies      []*IamInlinePolicy
	}

	IamPolicy struct {
		Name          string
		ConstructsRef core.AnnotationKeySet
		Policy        *PolicyDocument
	}

	IamInlinePolicy struct {
		Name          string
		ConstructsRef core.AnnotationKeySet
		Policy        *PolicyDocument
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
		ConstructsRef core.AnnotationKeySet
		ClientIdLists []string
		Cluster       *EksCluster
		Region        *Region
	}

	RolePolicyAttachment struct {
		Name          string
		ConstructsRef core.AnnotationKeySet
		Policy        *IamPolicy
		Role          *IamRole
	}

	InstanceProfile struct {
		Name          string
		ConstructsRef core.AnnotationKeySet
		Role          *IamRole
	}
)

type RoleCreateParams struct {
	AppName string
	Name    string
	Refs    core.AnnotationKeySet
}

func (role *IamRole) Create(dag *core.ResourceGraph, params RoleCreateParams) error {
	role.Name = roleSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	role.ConstructsRef = params.Refs.Clone()

	existingRole := dag.GetResource(role.Id())
	if existingRole != nil {
		return fmt.Errorf("iam role with name %s already exists", role.Name)
	}

	dag.AddResource(role)
	return nil
}

type IamPolicyCreateParams struct {
	AppName string
	Name    string
	Refs    core.AnnotationKeySet
}

func (policy *IamPolicy) Create(dag *core.ResourceGraph, params IamPolicyCreateParams) error {
	policy.Name = policySanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	policy.ConstructsRef = params.Refs.Clone()
	existingPolicy, found := core.GetResource[*IamPolicy](dag, policy.Id())
	if found {
		existingPolicy.ConstructsRef.AddAll(params.Refs)
		return nil
	}
	dag.AddResource(policy)
	return nil
}

type OidcCreateParams struct {
	AppName     string
	ClusterName string
	Refs        core.AnnotationKeySet
}

func (oidc *OpenIdConnectProvider) Create(dag *core.ResourceGraph, params OidcCreateParams) error {
	oidc.Name = fmt.Sprintf("%s-%s", params.AppName, params.ClusterName)

	existingOidc := dag.GetResource(oidc.Id())
	if existingOidc != nil {
		graphOidc := existingOidc.(*OpenIdConnectProvider)
		graphOidc.ConstructsRef.AddAll(params.Refs)
	} else {
		oidc.ConstructsRef = params.Refs.Clone()
		oidc.Region = NewRegion()
		subParams := map[string]any{
			"Cluster": EksClusterCreateParams{
				AppName: params.AppName,
				Name:    params.ClusterName,
				Refs:    params.Refs,
			},
		}
		err := dag.CreateDependencies(oidc, subParams)
		if err != nil {
			return err
		}
	}
	return nil
}

type InstanceProfileCreateParams struct {
	AppName string
	Name    string
	Refs    core.AnnotationKeySet
}

func (profile *InstanceProfile) Create(dag *core.ResourceGraph, params InstanceProfileCreateParams) error {
	profile.Name = roleSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	profile.ConstructsRef = params.Refs.Clone()
	existingProfile, found := core.GetResource[*InstanceProfile](dag, profile.Id())
	if found {
		existingProfile.ConstructsRef.AddAll(params.Refs)
		return nil
	}
	return dag.CreateDependencies(profile, map[string]any{
		"Role": params,
	})
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

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (role *IamRole) KlothoConstructRef() core.AnnotationKeySet {
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

func (role *IamRole) AddAwsManagedPolicies(policies []string) {
	currPolicies := map[string]bool{}
	for _, pol := range role.AwsManagedPolicies {
		currPolicies[pol] = true
	}
	for _, pol := range policies {
		if !currPolicies[pol] {
			role.AwsManagedPolicies = append(role.AwsManagedPolicies, pol)
			currPolicies[pol] = true
		}
	}
}

func (role *IamRole) AddManagedPolicy(policy core.IaCValue) {
	exists := false
	for _, pol := range role.ManagedPolicies {
		if pol == policy {
			exists = true
		}
	}
	if !exists {
		role.ManagedPolicies = append(role.ManagedPolicies, policy)
	}
}

func NewIamPolicy(appName string, policyName string, ref core.AnnotationKey, policy *PolicyDocument) *IamPolicy {
	return &IamPolicy{
		Name:          policySanitizer.Apply(fmt.Sprintf("%s-%s", appName, policyName)),
		ConstructsRef: core.AnnotationKeySetOf(ref),
		Policy:        policy,
	}
}

func NewIamInlinePolicy(policyName string, refs core.AnnotationKeySet, policy *PolicyDocument) *IamInlinePolicy {
	return &IamInlinePolicy{
		Name:          policySanitizer.Apply(policyName),
		ConstructsRef: refs,
		Policy:        policy,
	}
}

func (policy *IamPolicy) AddPolicyDocument(doc *PolicyDocument) {
	if policy.Policy == nil {
		policy.Policy = doc
		return
	}
	statement := doc.Statement
	policy.Policy.Statement = append(policy.Policy.Statement, statement...)
	policy.Policy.Deduplicate()
}

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (policy *IamPolicy) KlothoConstructRef() core.AnnotationKeySet {
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
func (oidc *OpenIdConnectProvider) KlothoConstructRef() core.AnnotationKeySet {
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

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (role *RolePolicyAttachment) KlothoConstructRef() core.AnnotationKeySet {
	return nil
}

// Id returns the id of the cloud resource
func (role *RolePolicyAttachment) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     IAM_ROLE_POLICY_ATTACHMENT_TYPE,
		Name:     role.Name,
	}
}

func (profile *InstanceProfile) KlothoConstructRef() core.AnnotationKeySet {
	return profile.ConstructsRef
}

// Id returns the id of the cloud resource
func (profile *InstanceProfile) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     INSTANCE_PROFILE_TYPE,
		Name:     profile.Name,
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
