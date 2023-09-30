package graphtest

import (
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateResourceId(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	unrelatedId := construct.ResourceId{Provider: "test", Name: "unrelated"}
	toUpdate := &construct.Resource{
		ID: construct.ResourceId{Provider: "test", Name: "updateMe"},
	}
	newProps := func() construct.Properties {
		return construct.Properties{
			"direct":       toUpdate.ID,
			"ref":          construct.PropertyRef{Resource: toUpdate.ID},
			"in array":     []any{toUpdate.ID},
			"in map value": map[string]any{"key": toUpdate.ID, "foo": "bar"},
			"in map key":   map[construct.ResourceId]struct{}{toUpdate.ID: {}, unrelatedId: {}},
		}
	}
	up := &construct.Resource{
		ID:         construct.ResourceId{Provider: "test", Name: "up"},
		Properties: newProps(),
	}
	down := &construct.Resource{
		ID:         construct.ResourceId{Provider: "test", Name: "down"},
		Properties: newProps(),
	}

	g := MakeGraph(t, construct.NewGraph(),
		toUpdate,
		up,
		down,
		"test::up -> test::updateMe",
		"test::updateMe -> test::down",
	)

	oldId := toUpdate.ID
	toUpdate.ID.Name = "updated"

	assert.Equal("test::updateMe", oldId.String())
	assert.Equal("test::updated", toUpdate.ID.String())

	err := construct.UpdateResourceId(g, oldId)
	require.NoError(err)

	newRes, err := g.Vertex(toUpdate.ID)
	require.NoError(err)
	assert.Equal(toUpdate.ID, newRes.ID)

	_, err = g.Vertex(oldId)
	assert.Error(err)

	assertProps := func(id construct.ResourceId) {
		res, err := g.Vertex(id)
		if !assert.NoError(err) {
			return
		}
		assert.Equal(
			toUpdate.ID,
			res.Properties["direct"],
		)
		assert.Equal(
			construct.PropertyRef{Resource: toUpdate.ID},
			res.Properties["ref"],
		)
		assert.Equal(
			[]any{toUpdate.ID},
			res.Properties["in array"],
		)
		assert.Equal(
			map[string]any{"key": toUpdate.ID, "foo": "bar"},
			res.Properties["in map value"],
		)
		assert.Equal(
			map[construct.ResourceId]struct{}{toUpdate.ID: {}, unrelatedId: {}},
			res.Properties["in map key"],
		)
	}

	upList, err := construct.DirectUpstreamDependencies(g, toUpdate.ID)
	if assert.NoError(err) {
		assert.Len(upList, 1)
		assert.Equal(up.ID, upList[0])
		assertProps(up.ID)
	}

	downList, err := construct.DirectDownstreamDependencies(g, toUpdate.ID)
	if assert.NoError(err) {
		assert.Len(downList, 1)
		assert.Equal(down.ID, downList[0])
		assertProps(down.ID)
	}
}

func TestRemoveResource(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	unrelatedId := construct.ResourceId{Provider: "test", Name: "unrelated"}
	toDelete := &construct.Resource{
		ID: construct.ResourceId{Provider: "test", Name: "deleteMe"},
	}
	newProps := func() construct.Properties {
		return construct.Properties{
			"direct":       toDelete.ID,
			"ref":          construct.PropertyRef{Resource: toDelete.ID},
			"in array":     []any{toDelete.ID},
			"in map value": map[string]any{"key": toDelete.ID, "foo": "bar"},
			"in map key":   map[construct.ResourceId]struct{}{toDelete.ID: {}, unrelatedId: {}},
		}
	}
	up := &construct.Resource{
		ID:         construct.ResourceId{Provider: "test", Name: "up"},
		Properties: newProps(),
	}
	down := &construct.Resource{
		ID:         construct.ResourceId{Provider: "test", Name: "down"},
		Properties: newProps(),
	}

	g := MakeGraph(t, construct.NewGraph(),
		toDelete,
		up,
		down,
		"test::up -> test::deleteMe",
		"test::deleteMe -> test::down",
		"test::up -> test::down",
	)

	err := construct.RemoveResource(g, toDelete.ID)
	require.NoError(err)

	_, err = g.Vertex(toDelete.ID)
	require.Error(err)

	assertProps := func(id construct.ResourceId) {
		res, err := g.Vertex(id)
		if !assert.NoError(err) {
			return
		}
		assert.Nil(res.Properties["direct"])
		assert.Nil(res.Properties["ref"])
		assert.Len(res.Properties["in array"], 0)
		assert.Equal(
			map[string]any{"foo": "bar"},
			res.Properties["in map value"],
		)
		assert.Equal(
			map[construct.ResourceId]struct{}{unrelatedId: {}},
			res.Properties["in map key"],
		)
	}

	downDeps, err := construct.DirectUpstreamDependencies(g, down.ID)
	if assert.NoError(err) {
		assert.Len(downDeps, 1)
		assert.Equal(up.ID, downDeps[0])
		assertProps(down.ID)
	}

	upDeps, err := construct.DirectDownstreamDependencies(g, up.ID)
	if assert.NoError(err) {
		assert.Len(upDeps, 1)
		assert.Equal(down.ID, upDeps[0])
		assertProps(up.ID)
	}
}
