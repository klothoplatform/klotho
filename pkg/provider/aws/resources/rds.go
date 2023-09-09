package resources

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"

	"github.com/klothoplatform/klotho/pkg/engine/classification"
	"github.com/klothoplatform/klotho/pkg/io"

	"github.com/klothoplatform/klotho/pkg/construct"
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

	rdsDBNameSanitizer = aws.RdsDBNameSanitizer
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
		ConstructRefs                    construct.BaseConstructSet `yaml:"-"`
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
		CredentialsFile                  io.File `yaml:"-"`
		CredentialsPath                  string
	}

	// RdsSubnetGroup represents an AWS RDS subnet group
	RdsSubnetGroup struct {
		Name          string
		ConstructRefs construct.BaseConstructSet `yaml:"-"`
		Subnets       []*Subnet
		Tags          map[string]string
	}

	// RdsProxy represents an AWS RDS proxy instance
	RdsProxy struct {
		Name              string
		ConstructRefs     construct.BaseConstructSet `yaml:"-"`
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
		SecretArn  construct.IaCValue
	}

	// RdsProxyTargetGroup represents an AWS RDS proxy target group
	RdsProxyTargetGroup struct {
		Name                            string
		ConstructRefs                   construct.BaseConstructSet `yaml:"-"`
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
	Refs    construct.BaseConstructSet
	Name    string
}

func (instance *RdsInstance) Create(dag *construct.ResourceGraph, params RdsInstanceCreateParams) error {

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

type RdsInstanceConfigureParams struct {
	DatabaseName string
}

func (instance *RdsInstance) Configure(params RdsInstanceConfigureParams) error {
	//TODO: enable this when we have a way to pass in the database name
	//instance.DatabaseName = params.DatabaseName
	instance.Username = generateUsername()
	instance.Password = generatePassword()
	credsBytes := []byte(fmt.Sprintf("{\n\"username\": \"%s\",\n\"password\": \"%s\"\n}", instance.Username, instance.Password))
	credsPath := fmt.Sprintf("secrets/%s", instance.Name)
	instance.CredentialsFile = &io.RawFile{
		FPath:   credsPath,
		Content: credsBytes,
	}
	instance.CredentialsPath = credsPath
	return nil
}

type RdsSubnetGroupCreateParams struct {
	AppName string
	Name    string
	Refs    construct.BaseConstructSet
}

func (subnetGroup *RdsSubnetGroup) Create(dag *construct.ResourceGraph, params RdsSubnetGroupCreateParams) error {
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

type RdsProxyCreateParams struct {
	AppName string
	Name    string
	Refs    construct.BaseConstructSet
}

func (proxy *RdsProxy) Create(dag *construct.ResourceGraph, params RdsProxyCreateParams) error {
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

type RdsProxyTargetGroupCreateParams struct {
	AppName string
	Name    string
	Refs    construct.BaseConstructSet
}

func (tg *RdsProxyTargetGroup) Create(dag *construct.ResourceGraph, params RdsProxyTargetGroupCreateParams) error {

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

func (rds *RdsInstance) GetConnectionPolicyDocument() *PolicyDocument {
	return CreateAllowPolicyDocument(
		[]string{"rds-db:connect"},
		[]construct.IaCValue{{ResourceId: rds.Id(), Property: RDS_CONNECTION_ARN_IAC_VALUE}})
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
func (rds *RdsInstance) BaseConstructRefs() construct.BaseConstructSet {
	return rds.ConstructRefs
}

// Id returns the id of the cloud resource
func (rds *RdsInstance) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     RDS_INSTANCE_TYPE,
		Name:     rds.Name,
	}
}
func (rds *RdsInstance) GetOutputFiles() []io.File {
	return []io.File{rds.CredentialsFile}
}

func (rds *RdsInstance) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream:     true,
		RequiresNoDownstream:   true,
		RequiresExplicitDelete: true,
	}
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (rds *RdsSubnetGroup) BaseConstructRefs() construct.BaseConstructSet {
	return rds.ConstructRefs
}

// Id returns the id of the cloud resource
func (rds *RdsSubnetGroup) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     RDS_SUBNET_GROUP_TYPE,
		Name:     rds.Name,
	}
}

func (rds *RdsSubnetGroup) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream: true,
	}
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (rds *RdsProxy) BaseConstructRefs() construct.BaseConstructSet {
	return rds.ConstructRefs
}

// Id returns the id of the cloud resource
func (rds *RdsProxy) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     RDS_PROXY_TYPE,
		Name:     rds.Name,
	}
}

func (rds *RdsProxy) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream: true,
	}
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (rds *RdsProxyTargetGroup) BaseConstructRefs() construct.BaseConstructSet {
	return rds.ConstructRefs
}

// Id returns the id of the cloud resource
func (rds *RdsProxyTargetGroup) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     RDS_PROXY_TARGET_GROUP,
		Name:     rds.Name,
	}
}

func (rds *RdsProxyTargetGroup) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstreamOrDownstream: true,
	}
}

func (rds *RdsInstance) MakeOperational(dag *construct.ResourceGraph, appName string, classifier *classification.ClassificationDocument) error {
	// Set a default database name to ensure we actually create a database on the instance
	if rds.DatabaseName == "" {
		rds.DatabaseName = rdsDBNameSanitizer.Apply(rds.Name)
	}
	return nil
}
