package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

const (
	KINESIS_STREAM_TYPE          = "kinesis_stream"
	KINESIS_STREAM_CONSUMER_TYPE = "kinesis_stream_consumer"
)

type (
	KinesisStream struct {
		Name                 string
		ConstructsRef        core.BaseConstructSet
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
		ConstructsRef core.BaseConstructSet
		ConsumerName  string
		Stream        *KinesisStream
	}
)

type KinesisStreamCreateParams struct {
	AppName string
	Refs    core.BaseConstructSet
	Name    string
}

func (stream *KinesisStream) Create(dag *core.ResourceGraph, params KinesisStreamCreateParams) error {

	name := aws.KinesisStreamSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	stream.Name = name
	stream.ConstructsRef = params.Refs

	existingStream, found := core.GetResource[*KinesisStream](dag, stream.Id())
	if found {
		existingStream.ConstructsRef.AddAll(params.Refs)
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

func (consumer *KinesisStreamConsumer) Create(dag *core.ResourceGraph, params KinesisStreamConsumerCreateParams) error {

	name := aws.KinesisStreamSanitizer.Apply(fmt.Sprintf("%s-%s", params.Stream.Name, params.Name))
	consumer.Name = name
	consumer.ConsumerName = aws.KinesisStreamSanitizer.Apply(params.Name)
	consumer.ConstructsRef = params.Stream.ConstructsRef.Clone()
	consumer.Stream = params.Stream
	existingConsumer, found := core.GetResource[*KinesisStreamConsumer](dag, consumer.Id())
	if found {
		existingConsumer.ConstructsRef.AddAll(params.Stream.ConstructsRef)
		return nil
	}
	dag.AddDependenciesReflect(consumer)
	return nil
}

// BaseConstructsRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (stream *KinesisStream) BaseConstructsRef() core.BaseConstructSet {
	return stream.ConstructsRef
}

// Id returns the id of the cloud resource
func (stream *KinesisStream) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     KINESIS_STREAM_TYPE,
		Name:     stream.Name,
	}
}

// BaseConstructsRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (consumer *KinesisStreamConsumer) BaseConstructsRef() core.BaseConstructSet {
	return consumer.ConstructsRef
}

// Id returns the id of the cloud resource
func (consumer *KinesisStreamConsumer) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     KINESIS_STREAM_CONSUMER_TYPE,
		Name:     consumer.Name,
	}
}
