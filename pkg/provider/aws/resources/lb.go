package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

var (
	loadBalancerSanitizer = aws.LoadBalancerSanitizer
	targetGroupSanitizer  = aws.TargetGroupSanitizer
)

const (
	LOAD_BALANCER_TYPE            = "load_balancer"
	TARGET_GROUP_TYPE             = "target_group"
	LISTENER_TYPE                 = "load_balancer_listener"
	NLB_INTEGRATION_URI_IAC_VALUE = "nlb_uri"
)

type (
	LoadBalancer struct {
		Name                   string
		ConstructsRef          []core.AnnotationKey
		IpAddressType          string
		LoadBalancerAttributes map[string]string
		Scheme                 string
		SecurityGroups         []*SecurityGroup
		Subnets                []*Subnet
		Tags                   map[string]string
		Type                   string
	}

	TargetGroup struct {
		Name          string
		ConstructsRef []core.AnnotationKey
		Port          int
		Protocol      string
		Vpc           *Vpc
		TargetType    string
		Tags          map[string]string
	}

	Listener struct {
		Name           string
		ConstructsRef  []core.AnnotationKey
		Port           int
		Protocol       string
		LoadBalancer   *LoadBalancer
		DefaultActions []*LBAction
	}
	LBAction struct {
		TargetGroupArn core.IaCValue
		Type           string
	}
)

func NewLoadBalancer(appName string, lbName string, refs []core.AnnotationKey, scheme string, lbType string, subnets []*Subnet, securityGroups []*SecurityGroup) *LoadBalancer {
	return &LoadBalancer{
		Name:           loadBalancerSanitizer.Apply(fmt.Sprintf("%s-%s", appName, lbName)),
		ConstructsRef:  refs,
		Scheme:         scheme,
		SecurityGroups: securityGroups,
		Subnets:        subnets,
		Type:           lbType,
	}
}

// Provider returns name of the provider the resource is correlated to
func (lb *LoadBalancer) Provider() string {
	return AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (lb *LoadBalancer) KlothoConstructRef() []core.AnnotationKey {
	return lb.ConstructsRef
}

// ID returns the id of the cloud resource
func (lb *LoadBalancer) Id() string {
	return fmt.Sprintf("%s:%s:%s", lb.Provider(), LOAD_BALANCER_TYPE, lb.Name)
}

func NewTargetGroup(appName string, tgName string, refs []core.AnnotationKey, port int, protocol string, vpc *Vpc, targetType string) *TargetGroup {
	return &TargetGroup{
		Name:          targetGroupSanitizer.Apply(fmt.Sprintf("%s-%s", appName, tgName)),
		ConstructsRef: refs,
		Port:          port,
		Protocol:      protocol,
		Vpc:           vpc,
		TargetType:    targetType,
	}
}

// Provider returns name of the provider the resource is correlated to
func (tg *TargetGroup) Provider() string {
	return AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (tg *TargetGroup) KlothoConstructRef() []core.AnnotationKey {
	return tg.ConstructsRef
}

// ID returns the id of the cloud resource
func (tg *TargetGroup) Id() string {
	return fmt.Sprintf("%s:%s:%s", tg.Provider(), TARGET_GROUP_TYPE, tg.Name)
}

func NewListener(name string, lb *LoadBalancer, ref []core.AnnotationKey, port int, protocol string, defaultActions []*LBAction) *Listener {
	return &Listener{
		Name:           targetGroupSanitizer.Apply(fmt.Sprintf("%s-%s", lb.Name, name)),
		ConstructsRef:  ref,
		Port:           port,
		Protocol:       protocol,
		LoadBalancer:   lb,
		DefaultActions: defaultActions,
	}
}

// Provider returns name of the provider the resource is correlated to
func (tg *Listener) Provider() string {
	return AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (tg *Listener) KlothoConstructRef() []core.AnnotationKey {
	return tg.ConstructsRef
}

// ID returns the id of the cloud resource
func (tg *Listener) Id() string {
	return fmt.Sprintf("%s:%s:%s", tg.Provider(), LISTENER_TYPE, tg.Name)
}
