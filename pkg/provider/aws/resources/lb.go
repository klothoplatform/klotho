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
	TARGET_GROUP_ARN_IAC_VALUE    = "target_group_arn"
)

type (
	LoadBalancer struct {
		Name                   string
		ConstructsRef          core.BaseConstructSet `yaml:"-"`
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
		ConstructsRef core.BaseConstructSet `yaml:"-"`
		Port          int
		Protocol      string
		Vpc           *Vpc
		TargetType    string
		Targets       []*Target
		Tags          map[string]string
	}

	Target struct {
		Id   *AwsResourceValue
		Port int
	}

	Listener struct {
		Name           string
		ConstructsRef  core.BaseConstructSet `yaml:"-"`
		Port           int
		Protocol       string
		LoadBalancer   *LoadBalancer
		DefaultActions []*LBAction
	}
	LBAction struct {
		TargetGroupArn *AwsResourceValue
		Type           string
	}
)

type LoadBalancerCreateParams struct {
	AppName     string
	Refs        core.BaseConstructSet
	Name        string
	NetworkType string
}

func (lb *LoadBalancer) Create(dag *core.ResourceGraph, params LoadBalancerCreateParams) error {
	lb.Name = loadBalancerSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	lb.ConstructsRef = params.Refs.Clone()

	existingLb, found := core.GetResource[*LoadBalancer](dag, lb.Id())
	if found {
		existingLb.ConstructsRef.AddAll(params.Refs)
		return nil
	}

	lb.Subnets = make([]*Subnet, 2)
	subnetType := PrivateSubnet
	if params.NetworkType == "public" {
		subnetType = PublicSubnet
	}
	subParams := map[string]any{
		"Subnets": []SubnetCreateParams{
			{
				AppName: params.AppName,
				Refs:    lb.ConstructsRef,
				AZ:      "0",
				Type:    subnetType,
			},
			{
				AppName: params.AppName,
				Refs:    lb.ConstructsRef,
				AZ:      "1",
				Type:    subnetType,
			},
		},
	}

	err := dag.CreateDependencies(lb, subParams)
	return err
}

type ListenerCreateParams struct {
	AppName     string
	Refs        core.BaseConstructSet
	Name        string
	NetworkType string
}

func (listener *Listener) Create(dag *core.ResourceGraph, params ListenerCreateParams) error {
	listener.Name = loadBalancerSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	listener.ConstructsRef = params.Refs.Clone()

	existingListener, found := core.GetResource[*Listener](dag, listener.Id())

	if found {
		existingListener.ConstructsRef.AddAll(params.Refs)
		return nil
	}

	err := dag.CreateDependencies(listener, map[string]any{
		"LoadBalancer": params,
	})
	return err
}

type TargetGroupCreateParams struct {
	AppName string
	Refs    core.BaseConstructSet
	Name    string
}

func (tg *TargetGroup) Create(dag *core.ResourceGraph, params TargetGroupCreateParams) error {
	tg.Name = targetGroupSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	tg.ConstructsRef = params.Refs.Clone()

	existingTg, found := core.GetResource[*TargetGroup](dag, tg.Id())
	if found {
		existingTg.ConstructsRef.AddAll(params.Refs)
		return nil
	}

	err := dag.CreateDependencies(tg, map[string]any{
		"Vpc": params,
	})
	return err
}

// BaseConstructsRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (lb *LoadBalancer) BaseConstructsRef() core.BaseConstructSet {
	return lb.ConstructsRef
}

// Id returns the id of the cloud resource
func (lb *LoadBalancer) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     LOAD_BALANCER_TYPE,
		Name:     lb.Name,
	}
}

func (lb *LoadBalancer) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstreamOrDownstream: true,
	}
}

func (tg *TargetGroup) AddTarget(target *Target) {
	addTarget := true
	for _, t := range tg.Targets {
		if t.Id.ResourceVal == target.Id.ResourceVal {
			addTarget = false
		}
	}
	if addTarget {
		tg.Targets = append(tg.Targets, target)
	}
}

// BaseConstructsRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (tg *TargetGroup) BaseConstructsRef() core.BaseConstructSet {
	return tg.ConstructsRef
}

// Id returns the id of the cloud resource
func (tg *TargetGroup) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     TARGET_GROUP_TYPE,
		Name:     tg.Name,
	}
}

func (tg *TargetGroup) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}

// BaseConstructsRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (listener *Listener) BaseConstructsRef() core.BaseConstructSet {
	return listener.ConstructsRef
}

// Id returns the id of the cloud resource
func (listener *Listener) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     LISTENER_TYPE,
		Name:     listener.Name,
	}
}

func (listener *Listener) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}
