package resources

import "github.com/klothoplatform/klotho/pkg/core"

type (
	SnsTopic struct {
		Name          string
		ConstructRefs core.BaseConstructSet `yaml:"-"`
		FifoTopic     bool
	}

	SnsSubscription struct {
		Name                string
		ConstructRefs       core.BaseConstructSet `yaml:"-"`
		Endpoint            core.IaCValue
		Protocol            string
		RawMessageDelivery  bool
		SubscriptionRoleArn *IamRole
		Topic               *SnsTopic
	}
)

const (
	SNS_TOPIC_TYPE        = "sns_topic"
	SNS_SUBSCRIPTION_TYPE = "sns_subscription"
)

func (q *SnsTopic) BaseConstructRefs() core.BaseConstructSet {
	return q.ConstructRefs
}

// Id returns the id of the cloud resource
func (q *SnsTopic) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     SNS_TOPIC_TYPE,
		Name:     q.Name,
	}
}

func (q *SnsTopic) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstreamOrDownstream: true,
	}
}

func (q *SnsSubscription) BaseConstructRefs() core.BaseConstructSet {
	return q.ConstructRefs
}

// Id returns the id of the cloud resource
func (q *SnsSubscription) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     SNS_SUBSCRIPTION_TYPE,
		Name:     q.Name,
	}
}

func (q *SnsSubscription) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstreamOrDownstream: true,
	}
}
