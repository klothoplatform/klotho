package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/construct/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_KinesisStreamCreate(t *testing.T) {
	eu := &types.ExecutionUnit{Name: "test"}
	eu2 := &types.ExecutionUnit{Name: "first"}
	initialRefs := construct.BaseConstructSetOf(eu2)
	cases := []coretesting.CreateCase[KinesisStreamCreateParams, *KinesisStream]{
		{
			Name: "nil kinesis stream",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:kinesis_stream:my-app-stream",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, record *KinesisStream) {
				assert.Equal(record.Name, "my-app-stream")
				assert.Equal(record.ConstructRefs, construct.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "existing kinesis stream",
			Existing: &KinesisStream{Name: "my-app-stream", ConstructRefs: initialRefs},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:kinesis_stream:my-app-stream",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, record *KinesisStream) {
				assert.Equal(record.Name, "my-app-stream")
				initialRefs.Add(eu)
				assert.Equal(record.ConstructRefs, initialRefs)
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = KinesisStreamCreateParams{
				Refs:    construct.BaseConstructSetOf(eu),
				AppName: "my-app",
				Name:    "stream",
			}
			tt.Run(t)
		})
	}
}

func Test_KinesisStreamConsumerCreate(t *testing.T) {
	eu := &types.ExecutionUnit{Name: "test"}
	eu2 := &types.ExecutionUnit{Name: "first"}
	initialRefs := construct.BaseConstructSetOf(eu2)
	cases := []coretesting.CreateCase[KinesisStreamConsumerCreateParams, *KinesisStreamConsumer]{
		{
			Name: "nil kinesis stream consumer",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:kinesis_stream_consumer:my-app-stream-consumer",
					"aws:kinesis_stream:my-app-stream",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:kinesis_stream_consumer:my-app-stream-consumer", Destination: "aws:kinesis_stream:my-app-stream"},
				},
			},
			Check: func(assert *assert.Assertions, record *KinesisStreamConsumer) {
				assert.Equal(record.Name, "my-app-stream-consumer")
				assert.Equal(record.ConsumerName, "consumer")
				assert.Equal(record.ConstructRefs, construct.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "existing kinesis stream consumer",
			Existing: &KinesisStreamConsumer{Name: "my-app-stream-consumer", ConstructRefs: initialRefs},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:kinesis_stream_consumer:my-app-stream-consumer",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, record *KinesisStreamConsumer) {
				assert.Equal(record.Name, "my-app-stream-consumer")
				initialRefs.Add(eu)
				assert.Equal(record.ConstructRefs, initialRefs)
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = KinesisStreamConsumerCreateParams{
				Stream: &KinesisStream{Name: "my-app-stream", ConstructRefs: construct.BaseConstructSetOf(eu)},
				Name:   "consumer",
			}
			tt.Run(t)
		})
	}
}
