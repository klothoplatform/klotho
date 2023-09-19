package engine

import (
	"fmt"
	"strings"

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
		Config   *knowledgebase.ConfigurationRule
	}
	Level string

	OutputDecision struct {
		Cause          Cause
		Resources      []construct.ResourceId
		Edges          []construct.OutputEdge
		Config         []knowledgebase.ConfigurationRule
		CauseMessage   string
		DisplayMessage string
	}
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

func (e *Engine) PostProcess(decisions []Decision) []OutputDecision {
	outputs := map[Cause]*OutputDecision{}
	for _, decision := range decisions {
		var output *OutputDecision
		var found bool
		output, found = outputs[*decision.Cause]
		if !found {
			output = &OutputDecision{Cause: *decision.Cause}
			outputs[*decision.Cause] = output
		}
		if decision.Result.Config != nil {
			output.Config = append(output.Config, *decision.Result.Config)
		} else if decision.Result.Resource != nil {
			output.Resources = append(output.Resources, decision.Result.Resource.Id())
		} else if decision.Result.Edge != nil {
			output.Edges = append(output.Edges, construct.OutputEdge{
				Source:      decision.Result.Edge.Source.Id(),
				Destination: decision.Result.Edge.Destination.Id(),
			})
		} else {
			panic("unknown decision result")
		}
	}
	outputDecisions := []OutputDecision{}
	for _, output := range outputs {
		output.DisplayMessage = output.String()
		outputDecisions = append(outputDecisions, *output)
	}
	return outputDecisions
}

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
	addResource := func(r construct.Resource, configure bool) bool {
		if context.ResourceGraph.GetResource(r.Id()) != nil {
			return false
		}
		context.ResourceGraph.AddResource(r)
		if configure {
			e.configureResource(context, r)
		}
		return true
	}
	switch decision.Action {
	case ActionConfigure:
		if decision.Result.Resource != nil && decision.Result.Config != nil {
			// Check to make sure the resource exists and if not log an error since we cannot configure
			if context.ResourceGraph.GetResource(decision.Result.Resource.Id()) == nil {
				context.Errors = append(context.Errors, &ResourceConfigurationError{
					Resource:   decision.Result.Resource,
					Config:     decision.Result.Config.Config,
					Constraint: decision.Cause.Constraint,
					Cause:      fmt.Errorf("resource %s not found in resource graph", decision.Result.Resource.Id()),
				})
				return
			}
			err := ConfigureField(decision.Result.Resource, decision.Result.Config.Config.Field, decision.Result.Config.Config.Value, decision.Result.Config.Config.ZeroValueAllowed, context.ResourceGraph)
			if err != nil {
				context.Errors = append(context.Errors, &InternalError{
					Cause: err,
					Child: &ResourceConfigurationError{
						Resource: decision.Result.Resource,
						Config:   decision.Result.Config.Config,
						Cause:    err,
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
			if addResource(decision.Result.Resource, true) {
				context.recordDecision(decision)
			}
		}
	case ActionConnect:
		if decision.Result.Edge != nil {
			addedSource := addResource(decision.Result.Edge.Source, false)
			addedDestination := addResource(decision.Result.Edge.Destination, false)

			if context.ResourceGraph.GetDependency(decision.Result.Edge.Source.Id(), decision.Result.Edge.Destination.Id()) == nil {
				context.ResourceGraph.AddDependencyWithData(decision.Result.Edge.Source, decision.Result.Edge.Destination, decision.Result.Edge.Properties.Data)
				context.recordDecision(decision)
			}
			// Configure the resources after the dependency was added
			if addedSource {
				e.configureResource(context, decision.Result.Edge.Source)
				context.recordDecision(Decision{Level: LevelInfo, Action: ActionCreate, Result: &DecisionResult{Resource: decision.Result.Edge.Source}, Cause: decision.Cause})
			}
			if addedDestination {
				e.configureResource(context, decision.Result.Edge.Destination)
				context.recordDecision(Decision{Level: LevelInfo, Action: ActionCreate, Result: &DecisionResult{Resource: decision.Result.Edge.Destination}, Cause: decision.Cause})
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

func (r DecisionResult) MarshalJSON() ([]byte, error) {
	if r.Config != nil {
		return []byte(`{"resource":"` + r.Resource.Id().String() + `","field":"` + r.Config.Config.Field + `","value":"` + fmt.Sprintf("%#v", r.Config.Config.Value) + `"}`), nil
	}
	if r.Resource != nil {
		return []byte(`{"resource":"` + r.Resource.Id().String() + `"}`), nil
	}
	if r.Edge != nil {
		return []byte(`{"edge":"` + r.Edge.Source.Id().String() + "," + r.Edge.Destination.Id().String() + `"}`), nil
	}
	return []byte("{}"), nil
}

func (c Cause) MarshalJSON() ([]byte, error) {
	if c.EdgeExpansion != nil {
		return []byte(`{"edge_expansion":"` + c.EdgeExpansion.Source.Id().String() + "," + c.EdgeExpansion.Destination.Id().String() + `"}`), nil
	}
	if c.EdgeConfiguration != nil {
		return []byte(`{"edge_configuration":"` + c.EdgeExpansion.Source.Id().String() + "," + c.EdgeExpansion.Destination.Id().String() + `"}`), nil
	}
	if c.OperationalResource != nil {
		return []byte(`{"operational_resource":"` + c.OperationalResource.Id().String() + `"}`), nil
	}
	if c.ResourceConfiguration != nil {
		return []byte(`{"resource_configuration":"` + c.ResourceConfiguration.Id().String() + `"}`), nil
	}
	if c.ConstructExpansion != nil {
		return []byte(`{"construct_expansion":"` + c.ConstructExpansion.Id().String() + `"}`), nil
	}
	if c.Constraint != nil {
		return []byte(`{"constraint":"` + c.Constraint.String() + `"}`), nil
	}
	return []byte("{}"), nil
}

func (d OutputDecision) String() string {
	if d.Cause.EdgeExpansion != nil {
		// generate a list of resources by the decisions edges
		pathResources := map[string]bool{}
		for _, resource := range d.Edges {
			pathResources[resource.Source.String()] = true
			pathResources[resource.Destination.String()] = true
		}
		pathString := ""
		for resource := range pathResources {
			pathString += fmt.Sprintf("	%s,", resource)
		}
		pathString = strings.TrimSuffix(pathString, ",")
		return fmt.Sprintf("Connected %s to %s, through the following resources: \n%s", d.Cause.EdgeExpansion.Source.Id().Name, d.Cause.EdgeExpansion.Destination.Id().Name, pathString)
	}
	if d.Cause.EdgeConfiguration != nil {
		var resourcesString string
		if len(d.Resources) > 0 {
			for i, resource := range d.Resources {
				if i < len(d.Resources)-2 {
					resourcesString += fmt.Sprintf(" %s,", resource.Name)
				} else {
					resourcesString += fmt.Sprintf(" %s", resource.Name)
				}
			}
			return fmt.Sprintf("connecting %s to %s caused the creation of: %s", d.Cause.EdgeConfiguration.Source.Id().Name, d.Cause.EdgeConfiguration.Destination.Id().Name, resourcesString)
		} else if len(d.Edges) > 0 {
			for _, edge := range d.Edges {
				resourcesString += fmt.Sprintf("	• %s -> %s\n", edge.Source.Name, edge.Destination.Name)
			}
			return fmt.Sprintf("connecting %s to %s caused the connections: %s", d.Cause.EdgeConfiguration.Source.Id().Name, d.Cause.EdgeConfiguration.Destination.Id().Name, resourcesString)
		}
	}
	if d.Cause.OperationalResource != nil {
		var resourcesString string
		if len(d.Resources) > 0 {
			for i, resource := range d.Resources {
				if i < len(d.Resources)-2 {
					resourcesString += fmt.Sprintf(" %s,", resource.Name)
				} else {
					resourcesString += fmt.Sprintf(" %s", resource.Name)
				}
			}
			return fmt.Sprintf("%s caused the creation of: %s", d.Cause.OperationalResource.Id().Name, resourcesString)
		}
	}
	if d.Cause.ConstructExpansion != nil {
		var resourcesString string
		if len(d.Resources) > 0 {
			for i, resource := range d.Resources {
				if i < len(d.Resources)-2 {
					resourcesString += fmt.Sprintf(" %s,", resource.Name)
				} else {
					resourcesString += fmt.Sprintf(" %s", resource.Name)
				}
			}
			return fmt.Sprintf("Expanding construct %s, created ", d.Cause.ConstructExpansion.Id().Name)
		}
	}
	return ""
}
