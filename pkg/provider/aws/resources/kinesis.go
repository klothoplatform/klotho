package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

const (
	KINESIS_STREAM_TYPE          = "kinesis_stream"
	KINESIS_STREAM_CONSUMER_TYPE = "kinesis_stream_consumer"
)

type (
	KinesisStream struct {
		Name                 string
		ConstructRefs        construct.BaseConstructSet `yaml:"-"`
		RetentionPeriodHours int
		ShardCount           int
		StreamEncryption     *StreamEncryption
		StreamModeDetails    StreamModeDetails
	}

	StreamEncryption struct {
		EncryptionType string
		Key            *KmsKey
	}

	StreamModeDetails struct {
		StreamMode string
	}

	KinesisStreamConsumer struct {
		Name          string
		ConstructRefs construct.BaseConstructSet `yaml:"-"`
		ConsumerName  string
		Stream        *KinesisStream
	}
)

type KinesisStreamCreateParams struct {
	AppName string
	Refs    construct.BaseConstructSet
	Name    string
}

func (stream *KinesisStream) Create(dag *construct.ResourceGraph, params KinesisStreamCreateParams) error {

	name := aws.KinesisStreamSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	stream.Name = name
	stream.ConstructRefs = params.Refs

	existingStream, found := construct.GetResource[*KinesisStream](dag, stream.Id())
	if found {
		existingStream.ConstructRefs.AddAll(params.Refs)
		return nil
	}
	dag.AddResource(stream)
	return nil
}

type KinesisStreamConfigureParams struct {
}

func (stream *KinesisStream) Configure(params KinesisStreamConfigureParams) error {
	stream.RetentionPeriodHours = 24
	stream.ShardCount = 1
	stream.StreamModeDetails = StreamModeDetails{StreamMode: "ON_DEMAND"}
	return nil
}

type KinesisStreamConsumerCreateParams struct {
	Stream *KinesisStream
	Name   string
}

func (consumer *KinesisStreamConsumer) Create(dag *construct.ResourceGraph, params KinesisStreamConsumerCreateParams) error {

	name := aws.KinesisStreamSanitizer.Apply(fmt.Sprintf("%s-%s", params.Stream.Name, params.Name))
	consumer.Name = name
	consumer.ConsumerName = aws.KinesisStreamSanitizer.Apply(params.Name)
	consumer.ConstructRefs = params.Stream.ConstructRefs.Clone()
	consumer.Stream = params.Stream
	existingConsumer, found := construct.GetResource[*KinesisStreamConsumer](dag, consumer.Id())
	if found {
		existingConsumer.ConstructRefs.AddAll(params.Stream.ConstructRefs)
		return nil
	}
	dag.AddDependenciesReflect(consumer)
	return nil
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (stream *KinesisStream) BaseConstructRefs() construct.BaseConstructSet {
	return stream.ConstructRefs
}

// Id returns the id of the cloud resource
func (stream *KinesisStream) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     KINESIS_STREAM_TYPE,
		Name:     stream.Name,
	}
}

func (role *KinesisStream) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstreamOrDownstream: true,
	}
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (consumer *KinesisStreamConsumer) BaseConstructRefs() construct.BaseConstructSet {
	return consumer.ConstructRefs
}

// Id returns the id of the cloud resource
func (consumer *KinesisStreamConsumer) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     KINESIS_STREAM_CONSUMER_TYPE,
		Name:     consumer.Name,
	}
}

func (consumer *KinesisStreamConsumer) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream: true,
	}
}
