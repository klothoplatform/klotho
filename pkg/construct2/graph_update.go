package construct2

import (
	"errors"

	"github.com/dominikbraun/graph"
	"github.com/klothoplatform/klotho/pkg/graph_addons"
)

func CopyVertexProps(p graph.VertexProperties) func(*graph.VertexProperties) {
	return func(dst *graph.VertexProperties) {
		*dst = p
	}
}

func CopyEdgeProps(p graph.EdgeProperties) func(*graph.EdgeProperties) {
	return func(dst *graph.EdgeProperties) {
		*dst = p
	}
}

// ReplaceResource replaces the resources identified by `oldId` with `newRes` in the graph and in any property
// references (as [ResourceId] or [PropertyRef]) of the old ID to the new ID in any resource that depends on or is
// depended on by the resource.
func ReplaceResource(g Graph, oldId ResourceId, newRes *Resource) error {
	err := graph_addons.ReplaceVertex(g, oldId, newRes, ResourceHasher)
	if err != nil {
		return err
	}

	updateId := func(path PropertyPathItem) error {
		itemVal := path.Get()

		if itemId, ok := itemVal.(ResourceId); ok && itemId == oldId {
			return path.Set(newRes.ID)
		}

		if itemRef, ok := itemVal.(PropertyRef); ok && itemRef.Resource == oldId {
			itemRef.Resource = newRes.ID
			return path.Set(itemRef)
		}
		return nil
	}

	return WalkGraph(g, func(id ResourceId, resource *Resource, nerr error) error {
		err = resource.WalkProperties(func(path PropertyPath, err error) error {
			err = errors.Join(err, updateId(path))

			if kv, ok := path.Last().(PropertyKVItem); ok {
				err = errors.Join(err, updateId(kv.Key()))
			}

			return err
		})
		return errors.Join(nerr, err)
	})
}

// UpdateResourceId is used when a resource's ID changes. It updates the graph in-place, using the resource
// currently referenced by `old`. No-op if the resource ID hasn't changed.
// Also updates any property references (as [ResourceId] or [PropertyRef]) of the old ID to the new ID in any
// resource that depends on or is depended on by the resource.
func PropagateUpdatedId(g Graph, old ResourceId) error {
	newRes, err := g.Vertex(old)
	if err != nil {
		return err
	}
	// Short circuit if the resource ID hasn't changed.
	if old == newRes.ID {
		return nil
	}
	return ReplaceResource(g, old, newRes)
}

// RemoveResource removes all edges from the resource. any property references (as [ResourceId] or [PropertyRef])
// to the resource, and finally the resource itself.
func RemoveResource(g Graph, id ResourceId) error {
	adj, err := g.AdjacencyMap()
	if err != nil {
		return err
	}
	if _, ok := adj[id]; !ok {
		return nil
	}

	for _, edge := range adj[id] {
		err = errors.Join(
			err,
			g.RemoveEdge(edge.Source, edge.Target),
		)
	}
	if err != nil {
		return err
	}

	pred, err := g.PredecessorMap()
	if err != nil {
		return err
	}
	for _, edge := range pred[id] {
		err = errors.Join(
			err,
			g.RemoveEdge(edge.Source, edge.Target),
		)
	}
	if err != nil {
		return err
	}

	removeId := func(path PropertyPathItem) (bool, error) {
		itemVal := path.Get()
		itemId, ok := itemVal.(ResourceId)
		if ok && itemId == id {
			return true, path.Remove(nil)

		}
		itemRef, ok := itemVal.(PropertyRef)
		if ok && itemRef.Resource == id {
			return true, path.Remove(nil)
		}
		return false, nil
	}

	for neighborId := range adj {
		neighbor, err := g.Vertex(neighborId)
		if err != nil {
			return err
		}
		err = neighbor.WalkProperties(func(path PropertyPath, nerr error) error {
			removed, err := removeId(path)
			nerr = errors.Join(nerr, err)
			if removed {
				return SkipProperty
			}
			kv, ok := path.Last().(PropertyKVItem)
			if !ok {
				return err
			}
			removed, err = removeId(kv.Key())
			nerr = errors.Join(nerr, err)
			if removed {
				return SkipProperty
			}
			return nerr
		})
		if err != nil {
			return err
		}
	}
	return g.RemoveVertex(id)
}
