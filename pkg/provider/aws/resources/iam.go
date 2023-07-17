package resources

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/engine/classification"
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
		ConstructRefs       core.BaseConstructSet `yaml:"-"`
		AssumeRolePolicyDoc *PolicyDocument
		ManagedPolicies     []core.IaCValue
		AwsManagedPolicies  []string
		InlinePolicies      []*IamInlinePolicy
	}

	IamPolicy struct {
		Name          string
		ConstructRefs core.BaseConstructSet `yaml:"-"`
		Policy        *PolicyDocument
	}

	IamInlinePolicy struct {
		Name          string
		ConstructRefs core.BaseConstructSet `yaml:"-"`
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
		ConstructRefs core.BaseConstructSet `yaml:"-"`
		ClientIdLists []string
		Cluster       *EksCluster
		Region        *Region
	}

	RolePolicyAttachment struct {
		Name          string
		ConstructRefs core.BaseConstructSet `yaml:"-"`
		Policy        *IamPolicy
		Role          *IamRole
	}

	InstanceProfile struct {
		Name          string
		ConstructRefs core.BaseConstructSet `yaml:"-"`
		Role          *IamRole
	}
)

type RoleCreateParams struct {
	AppName string
	Name    string
	Refs    core.BaseConstructSet
}

func (role *IamRole) Create(dag *core.ResourceGraph, params RoleCreateParams) error {
	role.Name = roleSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	role.ConstructRefs = params.Refs.Clone()

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
	Refs    core.BaseConstructSet
}

func (policy *IamPolicy) Create(dag *core.ResourceGraph, params IamPolicyCreateParams) error {
	policy.Name = policySanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	policy.ConstructRefs = params.Refs.Clone()
	existingPolicy, found := core.GetResource[*IamPolicy](dag, policy.Id())
	if found {
		existingPolicy.ConstructRefs.AddAll(params.Refs)
		return nil
	}
	dag.AddResource(policy)
	return nil
}

type OidcCreateParams struct {
	AppName     string
	ClusterName string
	Refs        core.BaseConstructSet
}

func (oidc *OpenIdConnectProvider) Create(dag *core.ResourceGraph, params OidcCreateParams) error {
	oidc.Name = fmt.Sprintf("%s-%s", params.AppName, params.ClusterName)

	existingOidc := dag.GetResource(oidc.Id())
	if existingOidc != nil {
		graphOidc := existingOidc.(*OpenIdConnectProvider)
		graphOidc.ConstructRefs.AddAll(params.Refs)
	} else {
		oidc.ConstructRefs = params.Refs.Clone()
		dag.AddResource(oidc)
	}
	return nil
}

func (oidc *OpenIdConnectProvider) MakeOperational(dag *core.ResourceGraph, appName string, classifier classification.Classifier) error {
	if oidc.Region == nil {
		oidc.Region = NewRegion()
	}
	if oidc.Cluster == nil {
		clusters := core.GetDownstreamResourcesOfType[*EksCluster](dag, oidc)
		if len(clusters) == 0 {
			return fmt.Errorf("oidc provider %s has no eks cluster downstream", oidc.Id())
		} else if len(clusters) == 1 {
			oidc.Cluster = clusters[0]
		} else {
			return fmt.Errorf("oidc provider %s has more than one eks cluster downstream", oidc.Id())
		}
	}
	dag.AddDependenciesReflect(oidc)
	return nil
}

type InstanceProfileCreateParams struct {
	AppName string
	Name    string
	Refs    core.BaseConstructSet
}

func (profile *InstanceProfile) Create(dag *core.ResourceGraph, params InstanceProfileCreateParams) error {
	profile.Name = roleSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	profile.ConstructRefs = params.Refs.Clone()
	existingProfile, found := core.GetResource[*InstanceProfile](dag, profile.Id())
	if found {
		existingProfile.ConstructRefs.AddAll(params.Refs)
		return nil
	}
	dag.AddResource(profile)
	return nil
}

func (profile *InstanceProfile) MakeOperational(dag *core.ResourceGraph, appName string, classifier classification.Classifier) error {
	if profile.Role == nil {
		roles := core.GetDownstreamResourcesOfType[*IamRole](dag, profile)
		if len(roles) == 0 {
			role, err := core.CreateResource[*IamRole](dag, RoleCreateParams{
				AppName: appName,
				Name:    fmt.Sprintf("%s-InstanceProfileRole", profile.Name),
				Refs:    core.BaseConstructSetOf(profile),
			})
			if err != nil {
				return err
			}
			profile.Role = role
			dag.AddDependency(profile, role)
		} else if len(roles) == 1 {
			profile.Role = roles[0]
			dag.AddDependenciesReflect(profile)
		} else {
			return fmt.Errorf("instance profile %s has more than one role downstream", profile.Id())
		}
	}
	return nil
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

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (role *IamRole) BaseConstructRefs() core.BaseConstructSet {
	return role.ConstructRefs
}

// Id returns the id of the cloud resource
func (role *IamRole) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     IAM_ROLE_TYPE,
		Name:     role.Name,
	}
}

func (role *IamRole) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
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
		if pol.ResourceId == policy.ResourceId {
			exists = true
		}
	}
	if !exists {
		role.ManagedPolicies = append(role.ManagedPolicies, policy)
	}
}

func NewIamPolicy(appName string, policyName string, ref core.BaseConstruct, policy *PolicyDocument) *IamPolicy {
	return &IamPolicy{
		Name:          policySanitizer.Apply(fmt.Sprintf("%s-%s", appName, policyName)),
		ConstructRefs: core.BaseConstructSetOf(ref),
		Policy:        policy,
	}
}

func NewIamInlinePolicy(policyName string, refs core.BaseConstructSet, policy *PolicyDocument) *IamInlinePolicy {
	return &IamInlinePolicy{
		Name:          policySanitizer.Apply(policyName),
		ConstructRefs: refs,
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

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (policy *IamPolicy) BaseConstructRefs() core.BaseConstructSet {
	return policy.ConstructRefs
}

// Id returns the id of the cloud resource
func (policy *IamPolicy) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     IAM_POLICY_TYPE,
		Name:     policy.Name,
	}
}

func (policy *IamPolicy) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (oidc *OpenIdConnectProvider) BaseConstructRefs() core.BaseConstructSet {
	return oidc.ConstructRefs
}

// Id returns the id of the cloud resource
func (oidc *OpenIdConnectProvider) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     OIDC_PROVIDER_TYPE,
		Name:     oidc.Name,
	}
}

func (oidc *OpenIdConnectProvider) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (role *RolePolicyAttachment) BaseConstructRefs() core.BaseConstructSet {
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

func (role *RolePolicyAttachment) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstreamOrDownstream: true,
	}
}

func (profile *InstanceProfile) BaseConstructRefs() core.BaseConstructSet {
	return profile.ConstructRefs
}

// Id returns the id of the cloud resource
func (profile *InstanceProfile) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     INSTANCE_PROFILE_TYPE,
		Name:     profile.Name,
	}
}

func (profile *InstanceProfile) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}

func (s StatementEntry) Id() core.ResourceId {
	resourcesHash := sha256.New()
	for _, r := range s.Resource {
		_, _ = fmt.Fprintf(resourcesHash, "%s.%s", r.ResourceId, r.Property)
	}

	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     IAM_STATEMENT_ENTRY,
		Name:     fmt.Sprintf("%x/%s/%s", resourcesHash.Sum(nil), s.Effect, strings.Join(s.Action, ",")),
	}
}

func (c Condition) MarshalYAML() (interface{}, error) {
	type mapEntry struct {
		Key core.IaCValue
		Val string
	}
	type condition struct {
		StringEquals []mapEntry
		Null         []mapEntry
	}
	intermediate := condition{}
	for k, v := range c.StringEquals {
		intermediate.StringEquals = append(intermediate.StringEquals, mapEntry{k, v})
	}
	for k, v := range c.Null {
		intermediate.Null = append(intermediate.Null, mapEntry{k, v})
	}
	return intermediate, nil
}

func (c *Condition) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type mapEntry struct {
		key core.IaCValue
		val string
	}
	type condition struct {
		StringEquals []mapEntry
		Null         []mapEntry
	}
	intermediate := condition{}
	err := unmarshal(&intermediate)
	if err != nil {
		return err
	}
	c.StringEquals = map[core.IaCValue]string{}
	c.Null = map[core.IaCValue]string{}
	for _, entry := range intermediate.StringEquals {
		c.StringEquals[entry.key] = entry.val
	}
	for _, entry := range intermediate.Null {
		c.Null[entry.key] = entry.val
	}
	return nil
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
