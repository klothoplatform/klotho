package constructs

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/async"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	"github.com/klothoplatform/klotho/pkg/k2/model"
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
	for _, c := range constraintList {
		if _, ok := c.(*constraints.ApplicationConstraint); ok {
			foundMustExist = true
		}
	}
	if !foundMustExist {
		t.Error("Expected to find at least one ApplicationConstraint with 'must_exist' operator")
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
