package constructs

import (
	"reflect"
	"testing"

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
		constructs: &async.ConcurrentMap[model.URN, *Construct]{},
	}
	constructURN, _ := model.ParseURN("urn:accountid:project:dev::construct/klotho.aws.Bucket:my-bucket")
	mockConstruct := &Construct{
		URN: *constructURN,
		Edges: []*Edge{
			{
				From: ResourceRef{ResourceKey: "aws:s3:test:bucket"},
				To:   ResourceRef{ResourceKey: "aws:ec2:test:instance"},
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
	mockEvaluator.constructs.Set(*constructURN, mockConstruct)
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

				foundMustExist := false
				foundEdge := false
				foundResource := false
				for _, c := range constraintList {
					switch constraint := c.(type) {
					case *constraints.ApplicationConstraint:
						foundMustExist = constraint.Operator == "must_exist"
					case *constraints.EdgeConstraint:
						foundEdge = true
					case *constraints.ResourceConstraint:
						foundResource = constraint.Property == "prop1" || constraint.Property == "prop2"
					}
				}

				assert.True(t, foundMustExist, "Expected to find at least one ApplicationConstraint with 'must_exist' operator")
				assert.True(t, foundEdge, "Expected to find at least one EdgeConstraint")
				assert.True(t, foundResource, "Expected to find at least one ResourceConstraint for 'prop1' or 'prop2'")
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

				foundOutput := false
				for _, c := range constraintList {
					if oc, ok := c.(*constraints.OutputConstraint); ok && oc.Name == "output1" && oc.Value == "outputValue1" {
						foundOutput = true
						break
					}
				}
				assert.True(t, foundOutput, "Expected to find an OutputConstraint for 'output1'")
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
						From: ResourceRef{ResourceKey: "aws:s3:test:bucket"},
						To:   ResourceRef{ResourceKey: "aws:ec2:test:instance"},
						Data: construct.EdgeData{},
					},
					{
						From: ResourceRef{ResourceKey: "aws:ec2:test:instance"},
						To:   ResourceRef{ResourceKey: "aws:lambda:test:function"},
						Data: construct.EdgeData{},
					},
				},
			},
			validateResult: func(t *testing.T, constraintList []constraints.Constraint) {
				assert.NotEmpty(t, constraintList)

				edgeCount := 0
				for _, c := range constraintList {
					if _, ok := c.(*constraints.EdgeConstraint); ok {
						edgeCount++
					}
				}
				assert.Equal(t, 2, edgeCount, "Expected 2 EdgeConstraints")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEvaluator.constructs.Set(*constructURN, tt.mockConstruct)
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
					"key2": ResourceRef{
						ResourceKey:  "aws:s3_bucket:mybucket",
						Type:         ResourceRefTypeTemplate,
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
					ResourceRef{
						ResourceKey:  "aws:s3_bucket:mybucket",
						Type:         ResourceRefTypeTemplate,
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
					Field2 ResourceRef
				}{
					Field1: "value1",
					Field2: ResourceRef{
						ResourceKey:  "aws:s3_bucket:mybucket",
						Type:         ResourceRefTypeTemplate,
						ConstructURN: *constructURN,
					},
				},
			},
			want: &struct {
				Field1 string
				Field2 ResourceRef
			}{
				Field1: "value1",
				Field2: ResourceRef{
					ResourceKey:  "aws:s3_bucket:mybucket",
					Type:         ResourceRefTypeTemplate,
					ConstructURN: *constructURN,
				},
			},
			wantErr: false,
		},
		{
			name: "marshal nested struct with settable ResourceRef",
			args: args{
				o: testConstruct,
				rawVal: &struct {
					Field1 string
					Nested struct {
						Field2 ResourceRef
					}
				}{
					Field1: "value1",
					Nested: struct {
						Field2 ResourceRef
					}{
						Field2: ResourceRef{
							ResourceKey:  "aws:s3_bucket:mybucket",
							Type:         ResourceRefTypeTemplate,
							ConstructURN: *constructURN,
						},
					},
				},
			},
			want: &struct {
				Field1 string
				Nested struct {
					Field2 ResourceRef
				}
			}{
				Field1: "value1",
				Nested: struct {
					Field2 ResourceRef
				}{
					Field2: ResourceRef{
						ResourceKey:  "aws:s3_bucket:mybucket",
						Type:         ResourceRefTypeTemplate,
						ConstructURN: *constructURN,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "marshal interface with ResourceRef",
			args: args{
				o: testConstruct,
				rawVal: interface{}(ResourceRef{
					ResourceKey:  "aws:s3_bucket:mybucket",
					Type:         ResourceRefTypeTemplate,
					ConstructURN: *constructURN,
				}),
			},
			want: construct.ResourceId{
				Provider: "aws",
				Type:     "s3_bucket",
				Name:     "mybucket",
			},
			wantErr: false,
		},
		{
			name: "marshal unsupported type",
			args: args{
				o:      testConstruct,
				rawVal: func() {}, // Using a function type to trigger the default case
			},
			want:    func() {}, // Expecting the same unsupported type to be returned
			wantErr: false,
		},
		{
			name: "marshal nil pointer",
			args: args{
				o:      testConstruct,
				rawVal: (*ResourceRef)(nil),
			},
			want:    (*ResourceRef)(nil),
			wantErr: false,
		},
		{
			name: "marshal nil map",
			args: args{
				o:      testConstruct,
				rawVal: (map[string]ResourceRef)(nil),
			},
			want:    (map[string]ResourceRef)(nil),
			wantErr: false,
		},
		{
			name: "marshal nil slice",
			args: args{
				o:      testConstruct,
				rawVal: ([]ResourceRef)(nil),
			},
			want:    ([]ResourceRef)(nil),
			wantErr: false,
		},
		{
			name: "marshal invalid value",
			args: args{
				o:      testConstruct,
				rawVal: nil,
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "marshal zero value",
			args: args{
				o:      testConstruct,
				rawVal: struct{}{},
			},
			want:    struct{}{},
			wantErr: false,
		},
		{
			name: "marshal pointer to struct with unsettable ResourceRef",
			args: args{
				o: testConstruct,
				rawVal: &struct {
					Field1 string
					field2 ResourceRef
				}{
					Field1: "value1",
					field2: ResourceRef{
						ResourceKey:  "aws:s3_bucket:mybucket",
						Type:         ResourceRefTypeTemplate,
						ConstructURN: *constructURN,
					},
				},
			},
			want: &struct {
				Field1 string
				field2 ResourceRef
			}{
				Field1: "value1",
				field2: ResourceRef{
					ResourceKey:  "aws:s3_bucket:mybucket",
					Type:         ResourceRefTypeTemplate,
					ConstructURN: *constructURN,
				},
			},
			wantErr: false,
		},
		{
			name: "marshal pointer to interface with ResourceRef",
			args: args{
				o: testConstruct,
				rawVal: func() interface{} {
					val := ResourceRef{
						ResourceKey:  "aws:s3_bucket:mybucket",
						Type:         ResourceRefTypeTemplate,
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
			wantErr: false,
		},
		{
			name: "marshal pointer to ResourceRef",
			args: args{
				o: testConstruct,
				rawVal: &ResourceRef{
					ResourceKey:  "aws:s3_bucket:mybucket",
					Type:         ResourceRefTypeTemplate,
					ConstructURN: *constructURN,
				},
			},
			want: construct.ResourceId{
				Provider: "aws",
				Type:     "s3_bucket",
				Name:     "mybucket",
			},
			wantErr: false,
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
			wantErr: false,
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
			wantErr: false,
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
			wantErr: false,
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
			if (err != nil) != tt.wantErr {
				t.Errorf("ConstructMarshaller.marshalRefs() error = %v, wantErr %v", err, tt.wantErr)
				return
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
