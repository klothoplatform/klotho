package aws

// func Test_ExpandExecutionUnit(t *testing.T) {
// 	eu := &core.ExecutionUnit{Name: "test", DockerfilePath: "path"}
// 	cases := []struct {
// 		name          string
// 		unit          *core.ExecutionUnit
// 		chart         *kubernetes.HelmChart
// 		constructType string
// 		want          coretesting.ResourcesExpectation
// 	}{
// 		{
// 			name:          "single lambda exec unit",
// 			unit:          eu,
// 			constructType: "lambda_function",
// 			want: coretesting.ResourcesExpectation{
// 				Nodes: []string{
// 					"aws:ecr_image:my-app-test",
// 					"aws:ecr_repo:my-app",
// 					"aws:iam_role:my-app-test-ExecutionRole",
// 					"aws:lambda_function:my-app-test",
// 					"aws:log_group:my-app-test",
// 				},
// 				Deps: []coretesting.StringDep{
// 					{Source: "aws:ecr_image:my-app-test", Destination: "aws:ecr_repo:my-app"},
// 					{Source: "aws:lambda_function:my-app-test", Destination: "aws:ecr_image:my-app-test"},
// 					{Source: "aws:lambda_function:my-app-test", Destination: "aws:iam_role:my-app-test-ExecutionRole"},
// 					{Source: "aws:lambda_function:my-app-test", Destination: "aws:log_group:my-app-test"},
// 				},
// 			},
// 		},
// 	}
// 	for _, tt := range cases {
// 		t.Run(tt.name, func(t *testing.T) {
// 			assert := assert.New(t)
// 			dag := core.NewResourceGraph()
// 			if tt.chart != nil {
// 				dag.AddResource(tt.chart)
// 			}

// 			aws := AWS{
// 				AppName: "my-app",
// 			}
// 			mappedRes, err := aws.expandExecutionUnit(dag, tt.unit, tt.constructType, map[string]any{})

// 			if !assert.NoError(err) {
// 				return
// 			}
// 			tt.want.Assert(t, dag)
// 			assert.NotEmpty(mappedRes)
// 		})
// 	}
// }
