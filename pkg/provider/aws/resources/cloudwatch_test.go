package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_CloudwatchLogGroupCreate(t *testing.T) {
	initialRefs := core.AnnotationKeySetOf(core.AnnotationKey{ID: "first"})
	cases := []struct {
		name     string
		logGroup *LogGroup
		want     coretesting.ResourcesExpectation
	}{
		{
			name: "nil repo",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:log_group:my-app-log-group",
				},
				Deps: []coretesting.StringDep{},
			},
		},
		{
			name:     "existing repo",
			logGroup: &LogGroup{Name: "my-app-log-group", ConstructsRef: initialRefs},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:log_group:my-app-log-group",
				},
				Deps: []coretesting.StringDep{},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			if tt.logGroup != nil {
				dag.AddResource(tt.logGroup)
			}
			metadata := CloudwatchLogGroupCreateParams{
				AppName: "my-app",
				Refs:    core.AnnotationKeySetOf(core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}),
				Name:    "log-group",
			}

			logGroup := &LogGroup{}
			err := logGroup.Create(dag, metadata)

			if !assert.NoError(err) {
				return
			}

			tt.want.Assert(t, dag)

			graphLogGroup := dag.GetResourceByVertexId(logGroup.Id().String())
			logGroup = graphLogGroup.(*LogGroup)

			assert.Equal(logGroup.Name, "my-app-log-group")
			if tt.logGroup == nil {
				assert.Equal(logGroup.ConstructsRef, metadata.Refs)
			} else {
				assert.Equal(logGroup, tt.logGroup)
				expect := initialRefs.CloneWith(core.AnnotationKeySetOf(core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}))
				assert.Equal(logGroup.KlothoConstructRef(), expect)
			}
		})
	}
}
