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
)

func TestConstructMarshaller_Marshal(t *testing.T) {
	mockEvaluator := &ConstructEvaluator{
		constructs: &async.ConcurrentMap[model.URN, *Construct]{},
	}

	constructURN, _ := model.ParseURN("urn:accountid:project:dev::construct/klotho.aws.Bucket:my-bucket")
	mockConstruct := &Construct{
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
		},
		Edges: []*Edge{
			{
				From: ResourceRef{ResourceKey: "aws:s3:test:bucket"},
				To:   ResourceRef{ResourceKey: "aws:ec2:test:instance"},
				Data: construct.EdgeData{},
			},
		},
	}

	mockEvaluator.constructs.Set(*constructURN, mockConstruct)

	marshaller := ConstructMarshaller{ConstructEvaluator: mockEvaluator}

	constraintList, err := marshaller.Marshal(*constructURN)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	if len(constraintList) == 0 {
		t.Error("Expected non-empty constraint list")
	}

	// Example checks on the constraint list
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

	if !foundMustExist {
		t.Error("Expected to find at least one ApplicationConstraint with 'must_exist' operator")
	}

	if !foundEdge {
		t.Error("Expected to find at least one EdgeConstraint")
	}

	if !foundResource {
		t.Error("Expected to find at least one ResourceConstraint for 'prop1' or 'prop2'")
	}
}

func TestConstructMarshaller_MarshalWithOutput(t *testing.T) {
	mockEvaluator := &ConstructEvaluator{
		constructs: &async.ConcurrentMap[model.URN, *Construct]{},
	}

	constructURN, _ := model.ParseURN("urn:accountid:project:dev::construct/klotho.aws.Bucket:my-bucket")
	mockConstruct := &Construct{
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
	}

	mockEvaluator.constructs.Set(*constructURN, mockConstruct)

	marshaller := ConstructMarshaller{ConstructEvaluator: mockEvaluator}

	constraintList, err := marshaller.Marshal(*constructURN)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	if len(constraintList) == 0 {
		t.Error("Expected non-empty constraint list")
	}

	// Check for output constraints
	foundOutput := false
	for _, c := range constraintList {
		if oc, ok := c.(*constraints.OutputConstraint); ok && oc.Name == "output1" && oc.Value == "outputValue1" {
			foundOutput = true
			break
		}
	}
	if !foundOutput {
		t.Error("Expected to find an OutputConstraint for 'output1'")
	}
}

func TestConstructMarshaller_MarshalEdges(t *testing.T) {
	mockEvaluator := &ConstructEvaluator{
		constructs: &async.ConcurrentMap[model.URN, *Construct]{},
	}

	constructURN, _ := model.ParseURN("urn:accountid:project:dev::construct/klotho.aws.Bucket:my-bucket")
	mockConstruct := &Construct{
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
		},
		Edges: []*Edge{
			{
				From: ResourceRef{ResourceKey: "aws:s3:test:bucket"},
				To:   ResourceRef{ResourceKey: "aws:ec2:test:instance"},
				Data: construct.EdgeData{},
			},
		},
	}

	mockEvaluator.constructs.Set(*constructURN, mockConstruct)

	marshaller := ConstructMarshaller{ConstructEvaluator: mockEvaluator}

	constraintList, err := marshaller.Marshal(*constructURN)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	if len(constraintList) == 0 {
		t.Error("Expected non-empty constraint list")
	}

	// Check for the presence of an EdgeConstraint
	foundEdgeConstraint := false
	for _, c := range constraintList {
		if _, ok := c.(*constraints.EdgeConstraint); ok {
			foundEdgeConstraint = true
		}
	}
	if !foundEdgeConstraint {
		t.Error("Expected to find at least one EdgeConstraint")
	}
}

func TestConstructMarshaller_EmptyConstruct(t *testing.T) {
	mockEvaluator := &ConstructEvaluator{
		constructs: &async.ConcurrentMap[model.URN, *Construct]{},
	}

	constructURN, _ := model.ParseURN("urn:accountid:project:dev::construct/klotho.aws.Empty:empty")
	mockConstruct := &Construct{
		URN:       *constructURN,
		Resources: map[string]*Resource{},
		Edges:     []*Edge{},
	}

	mockEvaluator.constructs.Set(*constructURN, mockConstruct)

	marshaller := ConstructMarshaller{ConstructEvaluator: mockEvaluator}

	constraintList, err := marshaller.Marshal(*constructURN)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	if len(constraintList) != 0 {
		t.Errorf("Expected empty constraint list, got %d constraints", len(constraintList))
	}
}

func TestConstructMarshaller_MultipleEdges(t *testing.T) {
	mockEvaluator := &ConstructEvaluator{
		constructs: &async.ConcurrentMap[model.URN, *Construct]{},
	}

	constructURN, _ := model.ParseURN("urn:accountid:project:dev::construct/klotho.aws.Bucket:my-bucket")
	mockConstruct := &Construct{
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
	}

	mockEvaluator.constructs.Set(*constructURN, mockConstruct)

	marshaller := ConstructMarshaller{ConstructEvaluator: mockEvaluator}

	constraintList, err := marshaller.Marshal(*constructURN)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	if len(constraintList) == 0 {
		t.Error("Expected non-empty constraint list")
	}

	// Check for the presence of multiple EdgeConstraints
	edgeCount := 0
	for _, c := range constraintList {
		if _, ok := c.(*constraints.EdgeConstraint); ok {
			edgeCount++
		}
	}
	if edgeCount != 2 {
		t.Errorf("Expected 2 EdgeConstraints, got %d", edgeCount)
	}
}

func TestConstructMarshaller_NestedResources(t *testing.T) {
	mockEvaluator := &ConstructEvaluator{
		constructs: &async.ConcurrentMap[model.URN, *Construct]{},
	}

	constructURN, _ := model.ParseURN("urn:accountid:project:dev::construct/klotho.aws.Bucket:my-bucket")
	mockConstruct := &Construct{
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
					"prop1": map[string]any{
						"nested1": "value1",
						"nested2": map[string]any{
							"subnested1": "subvalue1",
						},
					},
				},
			},
		},
	}

	mockEvaluator.constructs.Set(*constructURN, mockConstruct)

	marshaller := ConstructMarshaller{ConstructEvaluator: mockEvaluator}

	constraintList, err := marshaller.Marshal(*constructURN)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	if len(constraintList) == 0 {
		t.Error("Expected non-empty constraint list")
	}

	// Check for nested properties
	foundNested := false
	for _, c := range constraintList {
		if rc, ok := c.(*constraints.ResourceConstraint); ok {
			if nestedMap, ok := rc.Value.(map[string]any); ok {
				if subNestedMap, ok := nestedMap["nested2"].(map[string]any); ok {
					if _, exists := subNestedMap["subnested1"]; exists {
						foundNested = true
						break
					}
				}
			}
		}
	}
	if !foundNested {
		t.Error("Expected to find nested property constraint")
	}
}

func TestConstructMarshaller_OutputDeclarations(t *testing.T) {
	mockEvaluator := &ConstructEvaluator{
		constructs: &async.ConcurrentMap[model.URN, *Construct]{},
	}

	constructURN, _ := model.ParseURN("urn:accountid:project:dev::construct/klotho.aws.Bucket:my-bucket")
	mockConstruct := &Construct{
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
	}

	mockEvaluator.constructs.Set(*constructURN, mockConstruct)

	marshaller := ConstructMarshaller{ConstructEvaluator: mockEvaluator}

	constraintList, err := marshaller.Marshal(*constructURN)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	if len(constraintList) == 0 {
		t.Error("Expected non-empty constraint list")
	}

	// Check for the presence of OutputConstraint
	foundOutput := false
	for _, c := range constraintList {
		if _, ok := c.(*constraints.OutputConstraint); ok {
			foundOutput = true
		}
	}
	if !foundOutput {
		t.Error("Expected to find at least one OutputConstraint")
	}
}

func TestConstructMarshaller_MissingProperties(t *testing.T) {
	mockEvaluator := &ConstructEvaluator{
		constructs: &async.ConcurrentMap[model.URN, *Construct]{},
	}

	constructURN, _ := model.ParseURN("urn:accountid:project:dev::construct/klotho.aws.Bucket:my-bucket")
	mockConstruct := &Construct{
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
	}

	mockEvaluator.constructs.Set(*constructURN, mockConstruct)

	marshaller := ConstructMarshaller{ConstructEvaluator: mockEvaluator}

	constraintList, err := marshaller.Marshal(*constructURN)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	if len(constraintList) == 0 {
		t.Error("Expected non-empty constraint list")
	}

	// Check for missing properties handling
	foundMissingProp := false
	for _, c := range constraintList {
		if rc, ok := c.(*constraints.ResourceConstraint); ok && rc.Property == "prop2" {
			foundMissingProp = true
		}
	}
	if foundMissingProp {
		t.Error("Did not expect to find a constraint for missing property 'prop2'")
	}
}

func TestConstructMarshaller_marshalRefs_ConcreteTypes(t *testing.T) {
	type args struct {
		o      InfraOwner
		rawVal any
	}

	constructURN, _ := model.ParseURN("urn:example:construct::my-construct")
	testConstruct := &Construct{
		URN: *constructURN,
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
						ResourceKey:  "aws:s3:::bucket",
						Type:         ResourceRefTypeTemplate,
						ConstructURN: *constructURN,
					},
				},
			},
			want: map[string]any{
				"key1": "value1",
				"key2": ResourceRef{
					ResourceKey:  "aws:s3:::bucket",
					Type:         ResourceRefTypeTemplate,
					ConstructURN: *constructURN,
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
						ResourceKey:  "aws:s3:::bucket",
						Type:         ResourceRefTypeTemplate,
						ConstructURN: *constructURN,
					},
				},
			},
			want: []any{
				ResourceRef{
					ResourceKey:  "aws:s3:::bucket",
					Type:         ResourceRefTypeTemplate,
					ConstructURN: *constructURN,
				},
			},
			wantErr: false,
		},
		{
			name: "marshal struct with settable ResourceRef",
			args: args{
				o: testConstruct,
				rawVal: struct {
					Field1 string
					Field2 ResourceRef
				}{
					Field1: "value1",
					Field2: ResourceRef{
						ResourceKey:  "aws:s3:::bucket",
						Type:         ResourceRefTypeTemplate,
						ConstructURN: *constructURN,
					},
				},
			},
			want: struct {
				Field1 string
				Field2 ResourceRef
			}{
				Field1: "value1",
				Field2: ResourceRef{
					ResourceKey:  "aws:s3:::bucket",
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
				rawVal: struct {
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
							ResourceKey:  "aws:s3:::bucket",
							Type:         ResourceRefTypeTemplate,
							ConstructURN: *constructURN,
						},
					},
				},
			},
			want: struct {
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
						ResourceKey:  "aws:s3:::bucket",
						Type:         ResourceRefTypeTemplate,
						ConstructURN: *constructURN,
					},
				},
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
			if err != nil {
				t.Fatalf("Failed to create ConstructEvaluator: %v", err)
			}
			marshaller := &ConstructMarshaller{
				ConstructEvaluator: evaluator,
			}
			got, err := marshaller.marshalRefs(tt.args.o, tt.args.rawVal)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConstructMarshaller.marshalRefs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConstructMarshaller.marshalRefs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConstructMarshaller_marshalRefs_Pointers(t *testing.T) {
	type args struct {
		o      InfraOwner
		rawVal any
	}

	constructURN, _ := model.ParseURN("urn:example:construct::my-construct")
	testConstruct := &Construct{
		URN: *constructURN,
	}

	tests := []struct {
		name    string
		args    args
		want    any
		wantErr bool
	}{
		{
			name: "marshal pointer to struct with ResourceRef",
			args: args{
				o: testConstruct,
				rawVal: &struct {
					Field1 string
					Field2 ResourceRef
				}{
					Field1: "value1",
					Field2: ResourceRef{
						ResourceKey:  "aws:s3:::bucket",
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
					ResourceKey:  "aws:s3:::bucket",
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
						ResourceKey:  "aws:s3:::bucket",
						Type:         ResourceRefTypeTemplate,
						ConstructURN: *constructURN,
					}
					return &val
				}(),
			},
			want: func() interface{} {
				val := ResourceRef{
					ResourceKey:  "aws:s3:::bucket",
					Type:         ResourceRefTypeTemplate,
					ConstructURN: *constructURN,
				}
				return &val
			}(),
			wantErr: false,
		},
		{
			name: "marshal pointer to ResourceRef",
			args: args{
				o: testConstruct,
				rawVal: &ResourceRef{
					ResourceKey:  "aws:s3:::bucket",
					Type:         ResourceRefTypeTemplate,
					ConstructURN: *constructURN,
				},
			},
			want: &ResourceRef{
				ResourceKey:  "aws:s3:::bucket",
				Type:         ResourceRefTypeTemplate,
				ConstructURN: *constructURN,
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
			if err != nil {
				t.Fatalf("Failed to create ConstructEvaluator: %v", err)
			}
			marshaller := &ConstructMarshaller{
				ConstructEvaluator: evaluator,
			}
			got, err := marshaller.marshalRefs(tt.args.o, tt.args.rawVal)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConstructMarshaller.marshalRefs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConstructMarshaller.marshalRefs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConstructMarshaller_marshalRefs_Interfaces(t *testing.T) {
	type args struct {
		o      InfraOwner
		rawVal any
	}

	constructURN, _ := model.ParseURN("urn:example:construct::my-construct")
	testConstruct := &Construct{
		URN: *constructURN,
	}

	tests := []struct {
		name    string
		args    args
		want    any
		wantErr bool
	}{
		{
			name: "marshal interface with ResourceRef",
			args: args{
				o: testConstruct,
				rawVal: interface{}(ResourceRef{
					ResourceKey:  "aws:s3:::bucket",
					Type:         ResourceRefTypeTemplate,
					ConstructURN: *constructURN,
				}),
			},
			want: ResourceRef{
				ResourceKey:  "aws:s3:::bucket",
				Type:         ResourceRefTypeTemplate,
				ConstructURN: *constructURN,
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
			if err != nil {
				t.Fatalf("Failed to create ConstructEvaluator: %v", err)
			}
			marshaller := &ConstructMarshaller{
				ConstructEvaluator: evaluator,
			}
			got, err := marshaller.marshalRefs(tt.args.o, tt.args.rawVal)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConstructMarshaller.marshalRefs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConstructMarshaller.marshalRefs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConstructMarshaller_marshalRefs_UnsupportedType(t *testing.T) {
	type args struct {
		o      InfraOwner
		rawVal any
	}

	constructURN, _ := model.ParseURN("urn:example:construct::my-construct")
	testConstruct := &Construct{
		URN: *constructURN,
	}

	tests := []struct {
		name    string
		args    args
		want    any
		wantErr bool
	}{
		{
			name: "marshal unsupported type",
			args: args{
				o:      testConstruct,
				rawVal: func() {}, // Using a function type to trigger the default case
			},
			want:    func() {}, // Expecting the same unsupported type to be returned
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsys := afero.NewMemMapFs()
			stateManager := model.NewStateManager(fsys, "state.yaml")
			stackStateManager := stack.NewStateManager()
			evaluator, err := NewConstructEvaluator(stateManager, stackStateManager)
			if err != nil {
				t.Fatalf("Failed to create ConstructEvaluator: %v", err)
			}
			marshaller := &ConstructMarshaller{
				ConstructEvaluator: evaluator,
			}
			got, err := marshaller.marshalRefs(tt.args.o, tt.args.rawVal)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConstructMarshaller.marshalRefs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// Since we can't compare function types directly, we use reflection to check the type
			if reflect.TypeOf(got) != reflect.TypeOf(tt.want) {
				t.Errorf("ConstructMarshaller.marshalRefs() = %v, want %v", reflect.TypeOf(got), reflect.TypeOf(tt.want))
			}
		})
	}
}

func TestConstructMarshaller_marshalRefs_Nil(t *testing.T) {
	type args struct {
		o      InfraOwner
		rawVal any
	}

	constructURN, _ := model.ParseURN("urn:example:construct::my-construct")
	testConstruct := &Construct{
		URN: *constructURN,
	}

	tests := []struct {
		name    string
		args    args
		want    any
		wantErr bool
	}{
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsys := afero.NewMemMapFs()
			stateManager := model.NewStateManager(fsys, "state.yaml")
			stackStateManager := stack.NewStateManager()
			evaluator, err := NewConstructEvaluator(stateManager, stackStateManager)
			if err != nil {
				t.Fatalf("Failed to create ConstructEvaluator: %v", err)
			}
			marshaller := &ConstructMarshaller{
				ConstructEvaluator: evaluator,
			}
			got, err := marshaller.marshalRefs(tt.args.o, tt.args.rawVal)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConstructMarshaller.marshalRefs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConstructMarshaller.marshalRefs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConstructMarshaller_marshalRefs_Interpolated(t *testing.T) {
	type args struct {
		o      InfraOwner
		rawVal any
	}

	constructURN, _ := model.ParseURN("urn:example:construct::my-construct")
	testConstruct := &Construct{
		URN: *constructURN,
	}

	tests := []struct {
		name    string
		args    args
		want    any
		wantErr bool
	}{
		{
			name: "marshal ResourceRefTypeInterpolated",
			args: args{
				o: testConstruct,
				rawVal: ResourceRef{
					ResourceKey:  "interpolatedKey",
					Type:         ResourceRefTypeInterpolated,
					ConstructURN: *constructURN,
				},
			},
			want:    "interpolatedKey",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsys := afero.NewMemMapFs()
			stateManager := model.NewStateManager(fsys, "state.yaml")
			stackStateManager := stack.NewStateManager()
			evaluator, err := NewConstructEvaluator(stateManager, stackStateManager)
			if err != nil {
				t.Fatalf("Failed to create ConstructEvaluator: %v", err)
			}
			marshaller := &ConstructMarshaller{
				ConstructEvaluator: evaluator,
			}
			got, err := marshaller.marshalRefs(tt.args.o, tt.args.rawVal)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConstructMarshaller.marshalRefs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConstructMarshaller.marshalRefs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConstructMarshaller_marshalRefs_Invalid(t *testing.T) {
	type args struct {
		o      InfraOwner
		rawVal any
	}

	constructURN, _ := model.ParseURN("urn:example:construct::my-construct")
	testConstruct := &Construct{
		URN: *constructURN,
	}

	tests := []struct {
		name    string
		args    args
		want    any
		wantErr bool
	}{
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsys := afero.NewMemMapFs()
			stateManager := model.NewStateManager(fsys, "state.yaml")
			stackStateManager := stack.NewStateManager()
			evaluator, err := NewConstructEvaluator(stateManager, stackStateManager)
			if err != nil {
				t.Fatalf("Failed to create ConstructEvaluator: %v", err)
			}
			marshaller := &ConstructMarshaller{
				ConstructEvaluator: evaluator,
			}
			got, err := marshaller.marshalRefs(tt.args.o, tt.args.rawVal)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConstructMarshaller.marshalRefs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConstructMarshaller.marshalRefs() = %v, want %v", got, tt.want)
			}
		})
	}
}
