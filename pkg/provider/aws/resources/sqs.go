package resources

import "github.com/klothoplatform/klotho/pkg/construct"

type (
	SqsQueue struct {
		Name               string
		ConstructRefs      construct.BaseConstructSet `yaml:"-"`
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
		ConstructRefs  construct.BaseConstructSet `yaml:"-"`
		PolicyDocument *PolicyDocument
		Queues         []*SqsQueue
	}
)

const (
	SQS_QUEUE_TYPE        = "sqs_queue"
	SQS_QUEUE_POLICY_TYPE = "sqs_queue_policy"
)

func (q *SqsQueue) BaseConstructRefs() construct.BaseConstructSet {
	return q.ConstructRefs
}

// Id returns the id of the cloud resource
func (q *SqsQueue) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     SQS_QUEUE_TYPE,
		Name:     q.Name,
	}
}

func (q *SqsQueue) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstreamOrDownstream: true,
	}
}

func (q *SqsQueuePolicy) BaseConstructRefs() construct.BaseConstructSet {
	return q.ConstructRefs
}

// Id returns the id of the cloud resource
func (q *SqsQueuePolicy) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     SQS_QUEUE_POLICY_TYPE,
		Name:     q.Name,
	}
}

func (q *SqsQueuePolicy) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream: true,
	}
}
