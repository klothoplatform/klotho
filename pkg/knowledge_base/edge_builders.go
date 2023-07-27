package knowledgebase

import (
	"reflect"

	"github.com/klothoplatform/klotho/pkg/core"
)

type (
	EdgeBuilder[S core.Resource, D core.Resource] struct {
		Configure               typedEdgeFunc[S, D]
		DirectEdgeOnly          bool
		DeploymentOrderReversed bool
		DeletetionDependent     bool
		Reuse                   Reuse
	}

	typedEdgeFunc[S core.Resource, D core.Resource] func(source S, destination D, dag *core.ResourceGraph, data EdgeData) error

	edgeBuilder interface {
		Edge() Edge
		Details() EdgeDetails
	}
)

func (e EdgeBuilder[S, D]) Edge() Edge {
	var emptyS S
	var emptyD D
	return Edge{
		Source:      reflect.TypeOf(emptyS),
		Destination: reflect.TypeOf(emptyD),
	}
}

func (e EdgeBuilder[S, D]) Details() EdgeDetails {
	return EdgeDetails{
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data EdgeData) error {
			if e.Configure != nil {
				typedSource := source.(S)
				typedDest := dest.(D)
				return e.Configure(typedSource, typedDest, dag, data)
			}
			return nil
		},
		DirectEdgeOnly:          e.DirectEdgeOnly,
		DeploymentOrderReversed: e.DeploymentOrderReversed,
		DeletetionDependent:     e.DeletetionDependent,
		Reuse:                   e.Reuse,
	}
}

func Build(edges ...edgeBuilder) EdgeKB {
	result := make(EdgeMap)
	for _, builder := range edges {
		result[builder.Edge()] = builder.Details()
	}
	return NewEdgeKB(result)
}
