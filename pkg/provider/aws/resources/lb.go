package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/construct"
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
	TARGET_GROUP_ARN_IAC_VALUE    = "target_group_arn"
)

type (
	LoadBalancer struct {
		Name                   string
		ConstructRefs          construct.BaseConstructSet `yaml:"-"`
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
		ConstructRefs construct.BaseConstructSet `yaml:"-"`
		Port          int
		Protocol      string
		Vpc           *Vpc
		TargetType    string
		Targets       []*Target
		Tags          map[string]string
	}

	Target struct {
		Id   construct.IaCValue
		Port int
	}

	Listener struct {
		Name           string
		ConstructRefs  construct.BaseConstructSet `yaml:"-"`
		Port           int
		Protocol       string
		LoadBalancer   *LoadBalancer
		DefaultActions []*LBAction
	}
	LBAction struct {
		TargetGroupArn construct.IaCValue
		Type           string
	}
)

type LoadBalancerCreateParams struct {
	AppName string
	Refs    construct.BaseConstructSet
	Name    string
}

func (lb *LoadBalancer) Create(dag *construct.ResourceGraph, params LoadBalancerCreateParams) error {
	lb.Name = loadBalancerSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	lb.ConstructRefs = params.Refs.Clone()

	existingLb, found := construct.GetResource[*LoadBalancer](dag, lb.Id())
	if found {
		existingLb.ConstructRefs.AddAll(params.Refs)
		return nil
	}
	dag.AddResource(lb)
	return nil
}

type ListenerCreateParams struct {
	AppName     string
	Refs        construct.BaseConstructSet
	Name        string
	NetworkType string
}

func (listener *Listener) Create(dag *construct.ResourceGraph, params ListenerCreateParams) error {
	listener.Name = loadBalancerSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	listener.ConstructRefs = params.Refs.Clone()

	existingListener, found := construct.GetResource[*Listener](dag, listener.Id())

	if found {
		existingListener.ConstructRefs.AddAll(params.Refs)
		return nil
	}
	dag.AddResource(listener)
	return nil
}

type TargetGroupCreateParams struct {
	AppName string
	Refs    construct.BaseConstructSet
	Name    string
}

func (tg *TargetGroup) Create(dag *construct.ResourceGraph, params TargetGroupCreateParams) error {
	tg.Name = targetGroupSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	tg.ConstructRefs = params.Refs.Clone()

	existingTg, found := construct.GetResource[*TargetGroup](dag, tg.Id())
	if found {
		existingTg.ConstructRefs.AddAll(params.Refs)
		return nil
	}

	dag.AddResource(tg)
	return nil
}

func (tg *TargetGroup) SanitizedName() string {
	return targetGroupSanitizer.Apply(tg.Name)
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (lb *LoadBalancer) BaseConstructRefs() construct.BaseConstructSet {
	return lb.ConstructRefs
}

// Id returns the id of the cloud resource
func (lb *LoadBalancer) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     LOAD_BALANCER_TYPE,
		Name:     lb.Name,
	}
}

func (lb *LoadBalancer) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstreamOrDownstream: true,
	}
}

func (lb *LoadBalancer) SanitizedName() string {
	return loadBalancerSanitizer.Apply(lb.Name)
}

func (tg *TargetGroup) AddTarget(target *Target) {
	addTarget := true
	for _, t := range tg.Targets {
		if t.Id.ResourceId == target.Id.ResourceId {
			addTarget = false
		}
	}
	if addTarget {
		tg.Targets = append(tg.Targets, target)
	}
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (tg *TargetGroup) BaseConstructRefs() construct.BaseConstructSet {
	return tg.ConstructRefs
}

// Id returns the id of the cloud resource
func (tg *TargetGroup) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     TARGET_GROUP_TYPE,
		Name:     tg.Name,
	}
}

func (tg *TargetGroup) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream: true,
	}
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (listener *Listener) BaseConstructRefs() construct.BaseConstructSet {
	return listener.ConstructRefs
}

// Id returns the id of the cloud resource
func (listener *Listener) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     LISTENER_TYPE,
		Name:     listener.Name,
	}
}

func (listener *Listener) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream: true,
	}
}
