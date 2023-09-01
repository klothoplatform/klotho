package engine

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	"github.com/klothoplatform/klotho/pkg/graph"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
)

type (
	// Decision is a struct that represents a decision made by the engine
	Decision struct {
		Level  Level
		Result *DecisionResult
		Action Action
		Cause  *Cause
	}
	Action string
	Cause  struct {
		EdgeExpansion         *graph.Edge[construct.Resource]
		EdgeConfiguration     *graph.Edge[construct.Resource]
		OperationalResource   construct.Resource
		ResourceConfiguration construct.Resource
		ConstructExpansion    construct.BaseConstruct
		Constraint            constraints.Constraint
	}
	DecisionResult struct {
		Resource construct.Resource
		Edge     *graph.Edge[construct.Resource]
		Config   *knowledgebase.Configuration
	}
	Level string
)

const (
	// LevelError is the level of a decision when an error occurs
	LevelError Level = "error"
	// LevelInfo is the level of a decision when an informational decision is made
	LevelInfo Level = "info"
	// LevelDebug is the level of a decision when a debug decision is made
	LevelDebug Level = "debug"
	// LevelWarn is the level of a decision when a warning decision is made
	LevelWarn Level = "warn"

	ActionConfigure  Action = "configure"
	ActionDelete     Action = "delete"
	ActionCreate     Action = "create"
	ActionConnect    Action = "connect"
	ActionDisconnect Action = "disconnect"
)

func (context *SolveContext) recordDecision(decision Decision) {
	// Right now this is a wrapper around append, but we may want to do more in the future
	//
	// For example, we may want to check if there are duplicate decisions in the stack/history of decisions. If so we can likely determine that we are looping or going to fail and can shortcuircit the operations which are in a loop.
	context.Decisions = append(context.Decisions, decision)
}

// Here we want to validate that each decision is crafted correctly and that there is a valid cause, action and result pairing
func (d Decision) Validate() error {
	if d.Cause == nil {
		return fmt.Errorf("invalid decision cause")
	}
	if d.Result == nil {
		return fmt.Errorf("invalid decision result")
	}
	if d.Action == ActionConfigure && d.Result.Resource != nil && d.Result.Config != nil {
		return nil
	}
	if d.Action == ActionDelete && d.Result.Resource != nil {
		return nil
	}
	if d.Action == ActionCreate && d.Result.Resource != nil {
		return nil
	}
	if d.Action == ActionConnect && d.Result.Edge != nil {
		return nil
	}
	if d.Action == ActionDisconnect && d.Result.Edge != nil {
		return nil
	}
	return fmt.Errorf("invalid decision")

}

func (e *Engine) handleDecisions(context *SolveContext, decisions []Decision) {
	for _, decision := range decisions {
		err := decision.Validate()
		if err != nil {
			context.Errors = append(context.Errors, &InternalError{
				Cause: err,
			})
			continue
		}
		e.handleDecision(context, decision)
	}
}

func (e *Engine) handleDecision(context *SolveContext, decision Decision) {
	switch decision.Action {
	case ActionConfigure:
		if decision.Result.Resource != nil && decision.Result.Config != nil {
			// Check to make sure the resource exists and if not log an error since we cannot configure
			if context.ResourceGraph.GetResource(decision.Result.Resource.Id()) == nil {
				context.Errors = append(context.Errors, &ResourceConfigurationError{
					Resource:   decision.Result.Resource,
					Config:     *decision.Result.Config,
					Constraint: decision.Cause.Constraint,
					Cause:      fmt.Errorf("resource %s not found in resource graph", decision.Result.Resource.Id()),
				})
				return
			}
			err := ConfigureField(decision.Result.Resource, decision.Result.Config.Field, decision.Result.Config.Value, decision.Result.Config.ZeroValueAllowed, context.ResourceGraph)
			if err != nil {
				context.Errors = append(context.Errors, &InternalError{
					Cause: err,
					Child: &ResourceConfigurationError{
						Resource: decision.Result.Resource,
						Config:   *decision.Result.Config,
					},
				})
				return
			}
			context.recordDecision(decision)
		}
	case ActionDelete:
		if decision.Result.Resource != nil {
			if context.ResourceGraph.RemoveResource(decision.Result.Resource) == nil {
				err := context.ResourceGraph.RemoveResource(decision.Result.Resource)
				if err != nil {
					context.Errors = append(context.Errors, &InternalError{
						Cause: err,
					})
					return
				}
				context.recordDecision(decision)
			}
		}
	case ActionCreate:
		if decision.Result.Resource != nil {
			if context.ResourceGraph.GetResource(decision.Result.Resource.Id()) == nil {
				context.ResourceGraph.AddResource(decision.Result.Resource)
				context.recordDecision(decision)
			}
		}
	case ActionConnect:
		if decision.Result.Edge != nil {
			if context.ResourceGraph.GetResource(decision.Result.Edge.Source.Id()) == nil {
				e.handleDecision(context, Decision{Level: LevelInfo, Action: ActionCreate, Result: &DecisionResult{Resource: decision.Result.Edge.Source}, Cause: decision.Cause})
			}
			if context.ResourceGraph.GetResource(decision.Result.Edge.Destination.Id()) == nil {
				e.handleDecision(context, Decision{Level: LevelInfo, Action: ActionCreate, Result: &DecisionResult{Resource: decision.Result.Edge.Destination}, Cause: decision.Cause})
			}
			if context.ResourceGraph.GetDependency(decision.Result.Edge.Source.Id(), decision.Result.Edge.Destination.Id()) == nil {
				context.ResourceGraph.AddDependencyWithData(decision.Result.Edge.Source, decision.Result.Edge.Destination, decision.Result.Edge.Properties.Data)
				context.recordDecision(decision)
			}
		}
	case ActionDisconnect:
		if decision.Result.Edge != nil {
			if context.ResourceGraph.GetDependency(decision.Result.Edge.Source.Id(), decision.Result.Edge.Destination.Id()) != nil {
				err := context.ResourceGraph.RemoveDependency(decision.Result.Edge.Source.Id(), decision.Result.Edge.Destination.Id())
				if err != nil {
					context.Errors = append(context.Errors, &InternalError{
						Cause: err,
					})
					return
				}
				context.recordDecision(decision)
			}
		}
	default:
		panic("unknown action")
	}
}

// func (d Decision) MarshalJSON() ([]byte, error) {
// 	fmt.Println(`{"level":"` + string(d.Level) + `","result":"` + d.Result.String() + `","action":"` + string(d.Action) + `","cause":"` + d.Cause.String() + `"}`)
// 	return []byte(`{"level":"` + string(d.Level) + `","result":"` + d.Result.String() + `","action":"` + string(d.Action) + `","cause":"` + d.Cause.String() + `"}`), nil
// }

func (r DecisionResult) MarshalJSON() ([]byte, error) {
	if r.Config != nil {
		fmt.Println(`{"resource":"` + r.Resource.Id().String() + `","field":"` + r.Config.Field + `","value":` + fmt.Sprintf("%#v", r.Config.Value) + `"}`)
		return []byte(`{"resource":"` + r.Resource.Id().String() + `","field":"` + r.Config.Field + `","value":"` + fmt.Sprintf("%#v", r.Config.Value) + `"}`), nil
	}
	if r.Resource != nil {
		fmt.Println(`{"resource":"` + r.Resource.Id().String() + `"}`)
		return []byte(`{"resource":"` + r.Resource.Id().String() + `"}`), nil
	}
	if r.Edge != nil {
		fmt.Println(`{"edge":"` + r.Edge.Source.Id().String() + "," + r.Edge.Destination.Id().String() + `"}`)
		return []byte(`{"edge":"` + r.Edge.Source.Id().String() + "," + r.Edge.Destination.Id().String() + `"}`), nil
	}
	return []byte("{}"), nil
}

func (c Cause) MarshalJSON() ([]byte, error) {
	if c.EdgeExpansion != nil {
		fmt.Println(`{"edge_expansion":"` + c.EdgeExpansion.Source.Id().String() + "," + c.EdgeExpansion.Destination.Id().String() + `"}`)
		return []byte(`{"edge_expansion":"` + c.EdgeExpansion.Source.Id().String() + "," + c.EdgeExpansion.Destination.Id().String() + `"}`), nil
	}
	if c.EdgeConfiguration != nil {
		fmt.Println(`{"edge_configuration":"` + c.EdgeExpansion.Source.Id().String() + "," + c.EdgeExpansion.Destination.Id().String() + `"}`)
		return []byte(`{"edge_configuration":"` + c.EdgeExpansion.Source.Id().String() + "," + c.EdgeExpansion.Destination.Id().String() + `"}`), nil
	}
	if c.OperationalResource != nil {
		fmt.Println(`{"operational_resource":"` + c.OperationalResource.Id().String() + `"}`)
		return []byte(`{"operational_resource":"` + c.OperationalResource.Id().String() + `"}`), nil
	}
	if c.ResourceConfiguration != nil {
		fmt.Println(`{"resource_configuration":"` + c.ResourceConfiguration.Id().String() + `"}`)
		return []byte(`{"resource_configuration":"` + c.ResourceConfiguration.Id().String() + `"}`), nil
	}
	if c.ConstructExpansion != nil {
		fmt.Println(`{"construct_expansion":"` + c.ConstructExpansion.Id().String() + `"}`)
		return []byte(`{"construct_expansion":"` + c.ConstructExpansion.Id().String() + `"}`), nil
	}
	if c.Constraint != nil {
		fmt.Println(`constraint`)
		return []byte("constraint"), nil
	}
	return []byte("{}"), nil
}
