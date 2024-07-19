package constructs

import (
	"reflect"
	"testing"

	"github.com/klothoplatform/klotho/pkg/k2/constructs/template"
	"github.com/stretchr/testify/require"

	"github.com/klothoplatform/klotho/pkg/async"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/klothoplatform/klotho/pkg/k2/stack"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestConstructMarshaller(t *testing.T) {
	mockEvaluator := &ConstructEvaluator{
		Constructs: &async.ConcurrentMap[model.URN, *Construct]{},
	}
	constructURN, _ := model.ParseURN("urn:accountid:project:dev::construct/klotho.aws.Bucket:my-bucket")
	mockConstruct := &Construct{
		URN: *constructURN,
		Edges: []*Edge{
			{
				From: template.ResourceRef{ResourceKey: "aws:s3:test:bucket"},
				To:   template.ResourceRef{ResourceKey: "aws:ec2:test:instance"},
				Data: construct.EdgeData{},
			},
		},
		Resources: map[string]*Resource{
			"res1": {
				Id: construct.ResourceId{
					Provider:  "aws",
					Type:      "s3",
					Namespace: "test",
					Name:      "bucket",
				},
				Properties: construct.Properties{
					"prop1": "value1",
				},
			},
			"res2": {
				Id: construct.ResourceId{
					Provider:  "aws",
					Type:      "ec2",
					Namespace: "test",
					Name:      "instance",
				},
				Properties: construct.Properties{
					"prop2": "value2",
				},
			},
		},
	}
	mockEvaluator.Constructs.Set(*constructURN, mockConstruct)
	marshaller := ConstructMarshaller{ConstructEvaluator: mockEvaluator}

	tests := []struct {
		name           string
		mockConstruct  *Construct
		validateResult func(t *testing.T, constraintList []constraints.Constraint)
	}{
		{
			name:          "Marshal",
			mockConstruct: mockConstruct,
			validateResult: func(t *testing.T, constraintList []constraints.Constraint) {
				assert.NotEmpty(t, constraintList)

				constraints, err := constraints.ConstraintList(constraintList).ToConstraints()
				assert.NoError(t, err)

				assert.NotEmpty(t, constraints.Application, "Expected to find at least one ApplicationConstraint with 'must_exist' operator")
				assert.NotEmpty(t, constraints.Edges, "Expected to find at least one EdgeConstraint")
				assert.NotEmpty(t, constraints.Resources, "Expected to find at least one ResourceConstraint for 'prop1' or 'prop2'")
			},
		},
		{
			name: "MarshalWithOutput",
			mockConstruct: &Construct{
				URN: *constructURN,
				Resources: map[string]*Resource{
					"res1": {
						Id: construct.ResourceId{
							Provider:  "aws",
							Type:      "s3",
							Namespace: "test",
							Name:      "bucket",
						},
						Properties: construct.Properties{
							"prop1": "value1",
						},
					},
				},
				OutputDeclarations: map[string]OutputDeclaration{
					"output1": {
						Name:  "output1",
						Value: "outputValue1",
					},
				},
			},
			validateResult: func(t *testing.T, constraintList []constraints.Constraint) {
				assert.NotEmpty(t, constraintList)

				constraints, err := constraints.ConstraintList(constraintList).ToConstraints()
				assert.NoError(t, err)

				assert.NotEmpty(t, constraints.Outputs, "Expected to find an OutputConstraint for 'output1'")
			},
		},
		{
			name: "EmptyConstruct",
			mockConstruct: &Construct{
				URN:       *constructURN,
				Resources: map[string]*Resource{},
				Edges:     []*Edge{},
			},
			validateResult: func(t *testing.T, constraintList []constraints.Constraint) {
				assert.Empty(t, constraintList, "Expected empty constraint list")
			},
		},
		{
			name: "MultipleEdges",
			mockConstruct: &Construct{
				URN: *constructURN,
				Resources: map[string]*Resource{
					"res1": {
						Id: construct.ResourceId{
							Provider:  "aws",
							Type:      "s3",
							Namespace: "test",
							Name:      "bucket",
						},
						Properties: construct.Properties{
							"prop1": "value1",
						},
					},
					"res2": {
						Id: construct.ResourceId{
							Provider:  "aws",
							Type:      "ec2",
							Namespace: "test",
							Name:      "instance",
						},
						Properties: construct.Properties{
							"prop2": "value2",
						},
					},
					"res3": {
						Id: construct.ResourceId{
							Provider:  "aws",
							Type:      "lambda",
							Namespace: "test",
							Name:      "function",
						},
						Properties: construct.Properties{
							"prop3": "value3",
						},
					},
				},
				Edges: []*Edge{
					{
						From: template.ResourceRef{ResourceKey: "aws:s3:test:bucket"},
						To:   template.ResourceRef{ResourceKey: "aws:ec2:test:instance"},
						Data: construct.EdgeData{},
					},
					{
						From: template.ResourceRef{ResourceKey: "aws:ec2:test:instance"},
						To:   template.ResourceRef{ResourceKey: "aws:lambda:test:function"},
						Data: construct.EdgeData{},
					},
				},
			},
			validateResult: func(t *testing.T, constraintList []constraints.Constraint) {
				assert.NotEmpty(t, constraintList)

				constraints, err := constraints.ConstraintList(constraintList).ToConstraints()
				assert.NoError(t, err)

				assert.Equal(t, 2, len(constraints.Edges), "Expected 2 EdgeConstraints")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEvaluator.Constructs.Set(*constructURN, tt.mockConstruct)
			constraintList, err := marshaller.Marshal(*constructURN)
			assert.NoError(t, err, "Marshal() error = %v", err)
			tt.validateResult(t, constraintList)
		})
	}
}

func TestConstructMarshaller_marshalRefs(t *testing.T) {
	constructURN, _ := model.ParseURN("urn:example:construct::my-construct")
	testConstruct := &Construct{
		URN: *constructURN,
	}

	type args struct {
		o      InfraOwner
		rawVal any
	}

	tests := []struct {
		name    string
		args    args
		want    any
		wantErr bool
	}{
		{
			name: "marshal map with ResourceRef",
			args: args{
				o: testConstruct,
				rawVal: map[string]any{
					"key1": "value1",
					"key2": template.ResourceRef{
						ResourceKey:  "aws:s3_bucket:mybucket",
						Type:         template.ResourceRefTypeTemplate,
						ConstructURN: *constructURN,
					},
				},
			},
			want: map[string]any{
				"key1": "value1",
				"key2": construct.ResourceId{
					Provider: "aws",
					Type:     "s3_bucket",
					Name:     "mybucket",
				},
			},
			wantErr: false,
		},
		{
			name: "marshal slice with ResourceRef",
			args: args{
				o: testConstruct,
				rawVal: []any{
					template.ResourceRef{
						ResourceKey:  "aws:s3_bucket:mybucket",
						Type:         template.ResourceRefTypeTemplate,
						ConstructURN: *constructURN,
					},
				},
			},
			want: []any{
				construct.ResourceId{
					Provider: "aws",
					Type:     "s3_bucket",
					Name:     "mybucket",
				},
			},
			wantErr: false,
		},
		{
			name: "marshal struct with settable ResourceRef",
			args: args{
				o: testConstruct,
				rawVal: &struct {
					Field1 string
					Field2 template.ResourceRef
				}{
					Field1: "value1",
					Field2: template.ResourceRef{
						ResourceKey:  "aws:s3_bucket:mybucket",
						Type:         template.ResourceRefTypeTemplate,
						ConstructURN: *constructURN,
					},
				},
			},
			want: &struct {
				Field1 string
				Field2 template.ResourceRef
			}{
				Field1: "value1",
				Field2: template.ResourceRef{
					ResourceKey:  "aws:s3_bucket:mybucket",
					Type:         template.ResourceRefTypeTemplate,
					ConstructURN: *constructURN,
				},
			},
		},
		{
			name: "marshal nested struct with settable ResourceRef",
			args: args{
				o: testConstruct,
				rawVal: &struct {
					Field1 string
					Nested struct {
						Field2 template.ResourceRef
					}
				}{
					Field1: "value1",
					Nested: struct {
						Field2 template.ResourceRef
					}{
						Field2: template.ResourceRef{
							ResourceKey:  "aws:s3_bucket:mybucket",
							Type:         template.ResourceRefTypeTemplate,
							ConstructURN: *constructURN,
						},
					},
				},
			},
			want: &struct {
				Field1 string
				Nested struct {
					Field2 template.ResourceRef
				}
			}{
				Field1: "value1",
				Nested: struct {
					Field2 template.ResourceRef
				}{
					Field2: template.ResourceRef{
						ResourceKey:  "aws:s3_bucket:mybucket",
						Type:         template.ResourceRefTypeTemplate,
						ConstructURN: *constructURN,
					},
				},
			},
		},
		{
			name: "marshal interface with ResourceRef",
			args: args{
				o: testConstruct,
				rawVal: interface{}(template.ResourceRef{
					ResourceKey:  "aws:s3_bucket:mybucket",
					Type:         template.ResourceRefTypeTemplate,
					ConstructURN: *constructURN,
				}),
			},
			want: construct.ResourceId{
				Provider: "aws",
				Type:     "s3_bucket",
				Name:     "mybucket",
			},
		},
		{
			name: "marshal unsupported type",
			args: args{
				o:      testConstruct,
				rawVal: func() {}, // Using a function type to trigger the default case
			},
			want: func() {}, // Expecting the same unsupported type to be returned
		},
		{
			name: "marshal nil pointer",
			args: args{
				o:      testConstruct,
				rawVal: (*template.ResourceRef)(nil),
			},
			want: (*template.ResourceRef)(nil),
		},
		{
			name: "marshal nil map",
			args: args{
				o:      testConstruct,
				rawVal: (map[string]template.ResourceRef)(nil),
			},
			want: (map[string]template.ResourceRef)(nil),
		},
		{
			name: "marshal nil slice",
			args: args{
				o:      testConstruct,
				rawVal: ([]template.ResourceRef)(nil),
			},
			want: ([]template.ResourceRef)(nil),
		},
		{
			name: "marshal invalid value",
			args: args{
				o:      testConstruct,
				rawVal: nil,
			},
			want: nil,
		},
		{
			name: "marshal zero value",
			args: args{
				o:      testConstruct,
				rawVal: struct{}{},
			},
			want: struct{}{},
		},
		{
			name: "marshal pointer to struct with unsettable ResourceRef",
			args: args{
				o: testConstruct,
				rawVal: &struct {
					Field1 string
					field2 template.ResourceRef
				}{
					Field1: "value1",
					field2: template.ResourceRef{
						ResourceKey:  "aws:s3_bucket:mybucket",
						Type:         template.ResourceRefTypeTemplate,
						ConstructURN: *constructURN,
					},
				},
			},
			want: &struct {
				Field1 string
				field2 template.ResourceRef
			}{
				Field1: "value1",
				field2: template.ResourceRef{
					ResourceKey:  "aws:s3_bucket:mybucket",
					Type:         template.ResourceRefTypeTemplate,
					ConstructURN: *constructURN,
				},
			},
		},
		{
			name: "marshal pointer to interface with ResourceRef",
			args: args{
				o: testConstruct,
				rawVal: func() interface{} {
					val := template.ResourceRef{
						ResourceKey:  "aws:s3_bucket:mybucket",
						Type:         template.ResourceRefTypeTemplate,
						ConstructURN: *constructURN,
					}
					return &val
				}(),
			},
			want: construct.ResourceId{
				Provider: "aws",
				Type:     "s3_bucket",
				Name:     "mybucket",
			},
		},
		{
			name: "marshal pointer to ResourceRef",
			args: args{
				o: testConstruct,
				rawVal: &template.ResourceRef{
					ResourceKey:  "aws:s3_bucket:mybucket",
					Type:         template.ResourceRefTypeTemplate,
					ConstructURN: *constructURN,
				},
			},
			want: construct.ResourceId{
				Provider: "aws",
				Type:     "s3_bucket",
				Name:     "mybucket",
			},
		},
		{
			name: "marshal struct with unsettable int field",
			args: args{
				o: testConstruct,
				rawVal: &struct {
					Field1 string
					field2 int
				}{
					Field1: "value1",
					field2: 100,
				},
			},
			want: &struct {
				Field1 string
				field2 int
			}{
				Field1: "value1",
				field2: 100,
			},
		},
		{
			name: "marshal struct with settable int field",
			args: args{
				o: testConstruct,
				rawVal: &struct {
					Field1 string
					Field2 int
				}{
					Field1: "value1",
					Field2: 100,
				},
			},
			want: &struct {
				Field1 string
				Field2 int
			}{
				Field1: "value1",
				Field2: 100,
			},
		},
		{
			name: "marshal struct with pointer to int field",
			args: args{
				o: testConstruct,
				rawVal: &struct {
					Field1 string
					Field2 *int
				}{
					Field1: "value1",
					Field2: func() *int { v := 200; return &v }(),
				},
			},
			want: &struct {
				Field1 string
				Field2 *int
			}{
				Field1: "value1",
				Field2: func() *int { v := 200; return &v }(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsys := afero.NewMemMapFs()
			stateManager := model.NewStateManager(fsys, "state.yaml")
			stackStateManager := stack.NewStateManager()
			evaluator, err := NewConstructEvaluator(stateManager, stackStateManager)
			assert.NoError(t, err, "Failed to create ConstructEvaluator")
			marshaller := &ConstructMarshaller{
				ConstructEvaluator: evaluator,
			}
			got, err := marshaller.marshalRefs(tt.args.o, tt.args.rawVal)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			if tt.name == "marshal unsupported type" {
				// Since we can't compare function types directly, we use reflection to check the type
				if reflect.TypeOf(got) != reflect.TypeOf(tt.want) {
					t.Errorf("ConstructMarshaller.marshalRefs() = %v, want %v", reflect.TypeOf(got), reflect.TypeOf(tt.want))
				}
			} else {
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("ConstructMarshaller.marshalRefs() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}
