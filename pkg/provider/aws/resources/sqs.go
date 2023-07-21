package resources

import "github.com/klothoplatform/klotho/pkg/core"

type (
	SqsQueue struct {
		Name               string
		ConstructRefs      core.BaseConstructSet `yaml:"-"`
		FifoQueue          bool
		DelaySeconds       int
		MaximumMessageSize int
		RedrivePolicy      RedrivePolicy
		VisibilityTimeout  int
	}

	RedrivePolicy struct {
		DeadLetterTargetArn string
		MaxReceiveCount     int
	}

	SqsQueuePolicy struct {
		Name           string
		ConstructRefs  core.BaseConstructSet `yaml:"-"`
		PolicyDocument *PolicyDocument
		Queues         []*SqsQueue
	}
)

const (
	SQS_QUEUE_TYPE        = "sqs_queue"
	SQS_QUEUE_POLICY_TYPE = "sqs_queue_policy"
)

func (q *SqsQueue) BaseConstructRefs() core.BaseConstructSet {
	return q.ConstructRefs
}

// Id returns the id of the cloud resource
func (q *SqsQueue) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     SQS_QUEUE_TYPE,
		Name:     q.Name,
	}
}

func (q *SqsQueue) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstreamOrDownstream: true,
	}
}

func (q *SqsQueuePolicy) BaseConstructRefs() core.BaseConstructSet {
	return q.ConstructRefs
}

// Id returns the id of the cloud resource
func (q *SqsQueuePolicy) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     SQS_QUEUE_POLICY_TYPE,
		Name:     q.Name,
	}
}

func (q *SqsQueuePolicy) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}
