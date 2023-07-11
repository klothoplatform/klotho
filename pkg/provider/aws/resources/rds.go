package resources

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/engine/classification"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

var (
	RDS_ASSUME_ROLE_POLICY = &PolicyDocument{
		Version: VERSION,
		Statement: []StatementEntry{
			{
				Effect: "Allow",
				Principal: &Principal{
					Service: "rds.amazonaws.com",
				},
				Action: []string{"sts:AssumeRole"},
			},
		},
	}
	rdsInstanceSanitizer = aws.RdsInstanceSanitizer
	rdsSubnetSanitizer   = aws.RdsSubnetGroupSanitizer
	rdsProxySanitizer    = aws.RdsProxySanitizer
)

const (
	RDS_INSTANCE_TYPE      = "rds_instance"
	RDS_SUBNET_GROUP_TYPE  = "rds_subnet_group"
	RDS_PROXY_TYPE         = "rds_proxy"
	RDS_PROXY_TARGET_GROUP = "rds_proxy_target_group"

	RDS_CONNECTION_ARN_IAC_VALUE = "rds_connection_arn"
)

type (
	// RdsInstance represents an AWS RDS db instance
	RdsInstance struct {
		Name                             string
		ConstructRefs                    core.BaseConstructSet `yaml:"-"`
		SubnetGroup                      *RdsSubnetGroup
		SecurityGroups                   []*SecurityGroup
		DatabaseName                     string
		IamDatabaseAuthenticationEnabled bool
		Username                         string
		Password                         string
		Engine                           string
		EngineVersion                    string
		InstanceClass                    string
		SkipFinalSnapshot                bool
		AllocatedStorage                 int
		CredentialsFile                  core.File `yaml:"-"`
		CredentialsPath                  string
	}

	// RdsSubnetGroup represents an AWS RDS subnet group
	RdsSubnetGroup struct {
		Name          string
		ConstructRefs core.BaseConstructSet `yaml:"-"`
		Subnets       []*Subnet
		Tags          map[string]string
	}

	// RdsProxy represents an AWS RDS proxy instance
	RdsProxy struct {
		Name              string
		ConstructRefs     core.BaseConstructSet `yaml:"-"`
		DebugLogging      bool
		EngineFamily      string
		IdleClientTimeout int
		RequireTls        bool
		Role              *IamRole
		SecurityGroups    []*SecurityGroup
		Subnets           []*Subnet
		Auths             []*ProxyAuth `render:"document"`
	}

	// ProxyAuth represents an authorization configuration for an AWS RDS Proxy instance
	ProxyAuth struct {
		AuthScheme string
		IamAuth    string
		SecretArn  *AwsResourceValue
	}

	// RdsProxyTargetGroup represents an AWS RDS proxy target group
	RdsProxyTargetGroup struct {
		Name                            string
		ConstructRefs                   core.BaseConstructSet `yaml:"-"`
		RdsInstance                     *RdsInstance
		RdsProxy                        *RdsProxy
		TargetGroupName                 string
		ConnectionPoolConfigurationInfo *ConnectionPoolConfigurationInfo `render:"document"`
	}

	// ConnectionPoolConfigurationInfo represents the connection pool configuration within a RDS proxy target group
	ConnectionPoolConfigurationInfo struct {
		ConnectionBorrowTimeout   int
		InitQuery                 string
		MaxConnectionsPercent     int
		MaxIdleConnectionsPercent int
		SessionPinningFilters     []string
	}
)

type RdsInstanceCreateParams struct {
	AppName string
	Refs    core.BaseConstructSet
	Name    string
}

func (instance *RdsInstance) Create(dag *core.ResourceGraph, params RdsInstanceCreateParams) error {

	name := rdsInstanceSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	instance.Name = name
	instance.ConstructRefs = params.Refs.Clone()

	existingInstance := dag.GetResource(instance.Id())
	if existingInstance != nil {
		return fmt.Errorf("RdsInstance with name %s already exists", name)
	}
	dag.AddResource(instance)
	return nil
}

func (instance *RdsInstance) MakeOperational(dag *core.ResourceGraph, appName string, classifier classification.Classifier) error {
	var vpc *Vpc
	vpc, err := getSingleUpstreamVpc(dag, instance)
	if err != nil {
		return err
	}

	if instance.SubnetGroup == nil {
		subnetGroups := core.GetDownstreamResourcesOfType[*RdsSubnetGroup](dag, instance)
		if len(subnetGroups) > 1 {
			return fmt.Errorf("rds instance %s has multiple subnet group dependencies", instance.Name)
		} else if len(subnetGroups) == 0 {
			subnetGroup, err := core.CreateResource[*RdsSubnetGroup](dag, RdsSubnetGroupCreateParams{
				AppName: appName,
				Name:    fmt.Sprintf("%s-SubnetGroup", instance.Name),
				Refs:    core.BaseConstructSetOf(instance),
			})
			if err != nil {
				return err
			}
			instance.SubnetGroup = subnetGroup
			if vpc != nil {
				dag.AddDependency(subnetGroup, vpc)
			}
			err = subnetGroup.MakeOperational(dag, appName, classifier)
			if err != nil {
				return err
			}
		} else {
			instance.SubnetGroup = subnetGroups[0]
		}
	}

	if len(instance.SecurityGroups) == 0 {
		sgs, err := getSecurityGroupsOperational(dag, instance, appName)
		if err != nil {
			return err
		}
		instance.SecurityGroups = sgs
	}

	dag.AddDependenciesReflect(instance)
	return nil
}

type RdsInstanceConfigureParams struct {
	DatabaseName string
}

func (instance *RdsInstance) Configure(params RdsInstanceConfigureParams) error {
	instance.IamDatabaseAuthenticationEnabled = true
	instance.SkipFinalSnapshot = true
	instance.DatabaseName = params.DatabaseName
	instance.Username = generateUsername()
	instance.Password = generatePassword()

	instance.Engine = "postgres"
	instance.EngineVersion = "13.7"
	instance.InstanceClass = "db.t4g.micro"
	instance.AllocatedStorage = 20
	credsBytes := []byte(fmt.Sprintf("{\n\"username\": \"%s\",\n\"password\": \"%s\"\n}", instance.Username, instance.Password))
	credsPath := fmt.Sprintf("secrets/%s", instance.Name)
	instance.CredentialsFile = &core.RawFile{
		FPath:   credsPath,
		Content: credsBytes,
	}
	instance.CredentialsPath = credsPath
	return nil
}

type RdsSubnetGroupCreateParams struct {
	AppName string
	Name    string
	Refs    core.BaseConstructSet
}

func (subnetGroup *RdsSubnetGroup) Create(dag *core.ResourceGraph, params RdsSubnetGroupCreateParams) error {
	subnetGroup.Name = rdsSubnetSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	subnetGroup.ConstructRefs = params.Refs.Clone()

	existingSubnetGroup := dag.GetResource(subnetGroup.Id())
	if existingSubnetGroup != nil {
		graphSubnetGroup := existingSubnetGroup.(*RdsSubnetGroup)
		graphSubnetGroup.ConstructRefs.AddAll(params.Refs)
		return nil
	} else {
		dag.AddResource(subnetGroup)
	}
	return nil
}

func (subnetGroup *RdsSubnetGroup) MakeOperational(dag *core.ResourceGraph, appName string, classifier classification.Classifier) error {
	if len(subnetGroup.Subnets) == 0 {
		subnets, err := getSubnetsOperational(dag, subnetGroup, appName)
		if err != nil {
			return err
		}
		for _, subnet := range subnets {
			if subnet.Type == PrivateSubnet {
				subnetGroup.Subnets = append(subnetGroup.Subnets, subnet)
			}
		}
		dag.AddDependenciesReflect(subnetGroup)
	}
	return nil
}

type RdsProxyCreateParams struct {
	AppName string
	Name    string
	Refs    core.BaseConstructSet
}

func (proxy *RdsProxy) Create(dag *core.ResourceGraph, params RdsProxyCreateParams) error {
	proxy.Name = rdsSubnetSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	proxy.ConstructRefs = params.Refs.Clone()

	existingProxy := dag.GetResource(proxy.Id())
	if existingProxy != nil {
		graphProxy := existingProxy.(*RdsProxy)
		graphProxy.ConstructRefs.AddAll(params.Refs)
		return nil
	} else {
		dag.AddResource(proxy)
	}
	return nil
}
func (proxy *RdsProxy) MakeOperational(dag *core.ResourceGraph, appName string, classifier classification.Classifier) error {
	if proxy.Role == nil {
		roles := core.GetDownstreamResourcesOfType[*IamRole](dag, proxy)
		if len(roles) > 1 {
			return fmt.Errorf("rds proxy %s has multiple role dependencies", proxy.Name)
		} else if len(roles) == 0 {
			err := dag.CreateDependencies(proxy, map[string]any{
				"Role": RoleCreateParams{
					AppName: appName,
					Name:    fmt.Sprintf("%s-ProxyRole", proxy.Name),
					Refs:    proxy.ConstructRefs,
				},
			})
			if err != nil {
				return err
			}
		} else {
			proxy.Role = roles[0]
		}
	}

	if len(proxy.Subnets) == 0 {
		subnets, err := getSubnetsOperational(dag, proxy, appName)
		if err != nil {
			return err
		}
		for _, subnet := range subnets {
			if subnet.Type == PrivateSubnet {
				proxy.Subnets = append(proxy.Subnets, subnet)
			}
		}
	}

	if len(proxy.SecurityGroups) == 0 {
		sgs, err := getSecurityGroupsOperational(dag, proxy, appName)
		if err != nil {
			return err
		}
		proxy.SecurityGroups = sgs
	}

	dag.AddDependenciesReflect(proxy)
	return nil
}

type RdsProxyConfigureParams struct {
	EngineFamily      string
	DebugLogging      bool
	IdleClientTimeout int
	RequireTls        bool
}

func (proxy *RdsProxy) Configure(params RdsProxyConfigureParams) error {
	proxy.DebugLogging = false
	proxy.EngineFamily = "POSTGRESQL"
	proxy.IdleClientTimeout = 1800
	proxy.RequireTls = false
	return nil
}

type RdsProxyTargetGroupCreateParams struct {
	AppName string
	Name    string
	Refs    core.BaseConstructSet
}

func (tg *RdsProxyTargetGroup) Create(dag *core.ResourceGraph, params RdsProxyTargetGroupCreateParams) error {

	tg.Name = rdsProxySanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	tg.ConstructRefs = params.Refs.Clone()
	existingTG := dag.GetResource(tg.Id())
	if existingTG != nil {
		graphTG := existingTG.(*RdsProxyTargetGroup)
		graphTG.ConstructRefs.AddAll(params.Refs)
		return nil
	} else {
		dag.AddResource(tg)
	}
	return nil
}
func (tg *RdsProxyTargetGroup) MakeOperational(dag *core.ResourceGraph, appName string, classifier classification.Classifier) error {
	if tg.RdsProxy == nil {
		proxies := core.GetDownstreamResourcesOfType[*RdsProxy](dag, tg)
		if len(proxies) != 1 {
			return fmt.Errorf("rds proxy target group %s has %d proxy dependencies", tg.Name, len(proxies))
		}
		tg.RdsProxy = proxies[0]
		dag.AddDependency(proxies[0], tg)
	}
	if tg.RdsInstance == nil {
		instances := core.GetDownstreamResourcesOfType[*RdsInstance](dag, tg)
		if len(instances) != 1 {
			return fmt.Errorf("rds proxy target group %s has %d instance dependencies", tg.Name, len(instances))
		}
		tg.RdsInstance = instances[0]
		dag.AddDependency(tg, instances[0])
	}
	return nil
}

type RdsProxyTargetGroupConfigureParams struct {
	ConnectionPoolConfigurationInfo ConnectionPoolConfigurationInfo
}

// Configure sets the intristic characteristics of a vpc based on parameters passed in
func (targetGroup *RdsProxyTargetGroup) Configure(params RdsProxyTargetGroupConfigureParams) error {
	targetGroup.TargetGroupName = "default"
	targetGroup.ConnectionPoolConfigurationInfo = &ConnectionPoolConfigurationInfo{
		ConnectionBorrowTimeout:   120,
		MaxConnectionsPercent:     100,
		MaxIdleConnectionsPercent: 50,
	}
	return nil
}

func (rds *RdsInstance) GetConnectionPolicyDocument() *PolicyDocument {
	return CreateAllowPolicyDocument(
		[]string{"rds-db:connect"},
		[]*AwsResourceValue{{ResourceVal: rds, PropertyVal: RDS_CONNECTION_ARN_IAC_VALUE}})
}

// generateUsername generates a random username for the rds instance.
//
// The first letter of an RDS username must be a letter
func generateUsername() string {
	chars := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")
	length := 9
	var b strings.Builder
	b.WriteString("KLO")
	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			panic(err.Error())
		}
		b.WriteRune(chars[num.Int64()])
	}
	return b.String()
}

// generatePassword generates a random password for the rds instance.
func generatePassword() string {
	chars := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")
	length := 16
	var b strings.Builder
	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			panic(err.Error())
		}
		b.WriteRune(chars[num.Int64()])
	}
	return b.String()
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (rds *RdsInstance) BaseConstructRefs() core.BaseConstructSet {
	return rds.ConstructRefs
}

// Id returns the id of the cloud resource
func (rds *RdsInstance) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     RDS_INSTANCE_TYPE,
		Name:     rds.Name,
	}
}
func (rds *RdsInstance) GetOutputFiles() []core.File {
	return []core.File{rds.CredentialsFile}
}

func (rds *RdsInstance) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream:     true,
		RequiresNoDownstream:   true,
		RequiresExplicitDelete: true,
	}
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (rds *RdsSubnetGroup) BaseConstructRefs() core.BaseConstructSet {
	return rds.ConstructRefs
}

// Id returns the id of the cloud resource
func (rds *RdsSubnetGroup) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     RDS_SUBNET_GROUP_TYPE,
		Name:     rds.Name,
	}
}

func (rds *RdsSubnetGroup) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (rds *RdsProxy) BaseConstructRefs() core.BaseConstructSet {
	return rds.ConstructRefs
}

// Id returns the id of the cloud resource
func (rds *RdsProxy) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     RDS_PROXY_TYPE,
		Name:     rds.Name,
	}
}

func (rds *RdsProxy) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (rds *RdsProxyTargetGroup) BaseConstructRefs() core.BaseConstructSet {
	return rds.ConstructRefs
}

// Id returns the id of the cloud resource
func (rds *RdsProxyTargetGroup) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     RDS_PROXY_TARGET_GROUP,
		Name:     rds.Name,
	}
}

func (rds *RdsProxyTargetGroup) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstreamOrDownstream: true,
	}
}
