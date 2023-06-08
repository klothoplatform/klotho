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
		ConstructsRef          core.AnnotationKeySet
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
		ConstructsRef core.AnnotationKeySet
		Port          int
		Protocol      string
		Vpc           *Vpc
		TargetType    string
		Targets       []*Target
		Tags          map[string]string
	}

	Target struct {
		Id   core.IaCValue
		Port int
	}

	Listener struct {
		Name           string
		ConstructsRef  core.AnnotationKeySet
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

type LoadBalancerCreateParams struct {
	AppName     string
	Refs        core.AnnotationKeySet
	Name        string
	NetworkType string
}

func (lb *LoadBalancer) Create(dag *core.ResourceGraph, params LoadBalancerCreateParams) error {
	lb.Name = loadBalancerSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	lb.ConstructsRef = params.Refs

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
	Refs        core.AnnotationKeySet
	Name        string
	NetworkType string
}

func (listener *Listener) Create(dag *core.ResourceGraph, params ListenerCreateParams) error {
	listener.Name = loadBalancerSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	listener.ConstructsRef = params.Refs

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
	Refs    core.AnnotationKeySet
	Name    string
}

func (targetGroup *TargetGroup) Create(dag *core.ResourceGraph, params TargetGroupCreateParams) error {
	targetGroup.Name = targetGroupSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	targetGroup.ConstructsRef = params.Refs

	existingTg, found := core.GetResource[*TargetGroup](dag, targetGroup.Id())
	if found {
		existingTg.ConstructsRef.AddAll(params.Refs)
		return nil
	}

	err := dag.CreateDependencies(targetGroup, map[string]any{
		"Vpc": params,
	})
	return err
}

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (lb *LoadBalancer) KlothoConstructRef() core.AnnotationKeySet {
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

func (tg *TargetGroup) AddTarget(target *Target) {
	addTarget := true
	for _, t := range tg.Targets {
		if t.Id.Resource == target.Id.Resource {
			addTarget = false
		}
	}
	if addTarget {
		tg.Targets = append(tg.Targets, target)
	}
}

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (tg *TargetGroup) KlothoConstructRef() core.AnnotationKeySet {
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

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (tg *Listener) KlothoConstructRef() core.AnnotationKeySet {
	return tg.ConstructsRef
}

// Id returns the id of the cloud resource
func (tg *Listener) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     LISTENER_TYPE,
		Name:     tg.Name,
	}
}
