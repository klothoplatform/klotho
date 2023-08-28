package resources

import "github.com/klothoplatform/klotho/pkg/construct"

type (
	SnsTopic struct {
		Name          string
		ConstructRefs construct.BaseConstructSet `yaml:"-"`
		FifoTopic     bool
	}

	SnsSubscription struct {
		Name                string
		ConstructRefs       construct.BaseConstructSet `yaml:"-"`
		Endpoint            construct.IaCValue
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

func (q *SnsTopic) BaseConstructRefs() construct.BaseConstructSet {
	return q.ConstructRefs
}

// Id returns the id of the cloud resource
func (q *SnsTopic) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     SNS_TOPIC_TYPE,
		Name:     q.Name,
	}
}

func (q *SnsTopic) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstreamOrDownstream: true,
	}
}

func (q *SnsSubscription) BaseConstructRefs() construct.BaseConstructSet {
	return q.ConstructRefs
}

// Id returns the id of the cloud resource
func (q *SnsSubscription) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     SNS_SUBSCRIPTION_TYPE,
		Name:     q.Name,
	}
}

func (q *SnsSubscription) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstreamOrDownstream: true,
	}
}
