package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/construct/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_CloudwatchLogGroupCreate(t *testing.T) {
	eu := &types.ExecutionUnit{Name: "first"}
	eu2 := &types.ExecutionUnit{Name: "test"}
	initialRefs := construct.BaseConstructSetOf(eu)
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
			logGroup: &LogGroup{Name: "my-app-log-group", ConstructRefs: initialRefs},
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
			dag := construct.NewResourceGraph()
			if tt.logGroup != nil {
				dag.AddResource(tt.logGroup)
			}
			metadata := CloudwatchLogGroupCreateParams{
				AppName: "my-app",
				Refs:    construct.BaseConstructSetOf(eu2),
				Name:    "log-group",
			}

			logGroup := &LogGroup{}
			err := logGroup.Create(dag, metadata)

			if !assert.NoError(err) {
				return
			}

			tt.want.Assert(t, dag)

			graphLogGroup := dag.GetResource(logGroup.Id())
			logGroup = graphLogGroup.(*LogGroup)

			assert.Equal(logGroup.Name, "my-app-log-group")
			if tt.logGroup == nil {
				assert.Equal(logGroup.ConstructRefs, metadata.Refs)
			} else {
				assert.Equal(logGroup, tt.logGroup)
				expect := initialRefs.CloneWith(construct.BaseConstructSetOf(eu2))
				assert.Equal(logGroup.BaseConstructRefs(), expect)
			}
		})
	}
}
