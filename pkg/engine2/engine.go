package engine2

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

type (
	// Engine is a struct that represents the object which processes the resource graph and applies constraints
	Engine struct {
		Kb *knowledgebase.KnowledgeBase
		// The context of the engine
		Context EngineContext
	}

	// EngineContext is a struct that represents the context of the engine
	// The context is used to store the state of the engine
	EngineContext struct {
		Constraints  map[constraints.ConstraintScope][]constraints.Constraint
		InitialState *construct.ResourceGraph
	}
)

func (e *Engine) CreateResourceFromId(id construct.ResourceId) construct.Resource {
	panic("implement me")
}

func NewEngine(kb *knowledgebase.KnowledgeBase) *Engine {
	return &Engine{
		Kb: kb,
	}
}

func (e *Engine) Run() error {
	if e.Context.InitialState == nil {
		return errors.New("initial state is nil")
	}

	// // First we look at all application constraints to see what is going to be added and removed from the construct graph
	// for _, constraint := range e.Context.Constraints[constraints.ApplicationConstraintScope] {
	// 	err := e.ApplyApplicationConstraint(constraint.(*constraints.ApplicationConstraint))
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// }

	return nil
}

// ApplyApplicationConstraint applies an application constraint to the either the engines working state construct graph
//
// Currently ApplicationConstraints can only be applied if the representing nodes are klotho constructs and not provider level resources
func (e *Engine) ApplyApplicationConstraint(constraint *constraints.ApplicationConstraint, ctx solution_context.SolutionContext) error {
	switch constraint.Operator {
	case constraints.AddConstraintOperator:
		res := e.CreateResourceFromId(constraint.Node)
		ctx.AddResource(res)
	case constraints.RemoveConstraintOperator:
		node := ctx.GetResource(constraint.Node)
		if node == nil {
			return fmt.Errorf("could not find resource %s", constraint.Node)
		}
		return ctx.RemoveResource(node, true)
	case constraints.ReplaceConstraintOperator:
		node := ctx.GetResource(constraint.Node)
		if node == nil {
			return fmt.Errorf("could not find resource %s", constraint.Node)
		}
		var replacementNode construct.Resource
		if node.Id().QualifiedTypeName() == constraint.ReplacementNode.QualifiedTypeName() {
			replacementNode = cloneResource(node)
			reflect.ValueOf(replacementNode).Elem().FieldByName("Name").Set(reflect.ValueOf(constraint.ReplacementNode.Name))
		} else {
			replacementNode = e.CreateResourceFromId(constraint.ReplacementNode)
		}
		return ctx.ReplaceResourceId(constraint.Node, replacementNode)
	}
	return nil
}

func cloneResource(resource construct.Resource) construct.Resource {
	newRes := reflect.New(reflect.TypeOf(resource).Elem()).Interface().(construct.Resource)
	for i := 0; i < reflect.ValueOf(newRes).Elem().NumField(); i++ {
		field := reflect.ValueOf(newRes).Elem().Field(i)
		field.Set(reflect.ValueOf(resource).Elem().Field(i))
	}
	return newRes
}
