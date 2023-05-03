package aws

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/stretchr/testify/assert"
)

func TestAwsMapResourceDirectlyToConstruct(t *testing.T) {
	t.Run("empty AWS struct, MapResourceDirectlyToConstruct", func(t *testing.T) {
		assert := assert.New(t)
		a := AWS{}
		a.MapResourceDirectlyToConstruct(dummyResource("res"), dummyConstruct("cons"))
		assert.Equal(
			map[string][]core.Resource{
				"cons": {dummyResource("res")},
			},
			a.constructIdToResources)
	})
	t.Run("empty AWS struct, MapResourceDirectlyToConstruct", func(t *testing.T) {
		assert := assert.New(t)
		a := AWS{}
		_, found := a.GetResourcesDirectlyTiedToConstruct(dummyConstruct("cons"))
		assert.False(found)
	})
	t.Run("resources get sorted", func(t *testing.T) {
		assert := assert.New(t)
		a := AWS{}
		a.MapResourceDirectlyToConstruct(dummyResource("res_b"), dummyConstruct("cons"))
		a.MapResourceDirectlyToConstruct(dummyResource("res_a"), dummyConstruct("cons"))
		assert.Equal(
			map[string][]core.Resource{
				"cons": {dummyResource("res_a"), dummyResource("res_b")}, // note: NOT the order they were added above!
			},
			a.constructIdToResources)
	})
	t.Run("returns multiple resources", func(t *testing.T) {
		assert := assert.New(t)
		a := AWS{
			constructIdToResources: map[string][]core.Resource{
				"cons": {dummyResource("res_a"), dummyResource("res_b"), dummyResource("res_c")},
			},
		}
		resList, found := a.GetResourcesDirectlyTiedToConstruct(dummyConstruct("cons"))
		assert.True(found)
		assert.Equal(
			[]core.Resource{dummyResource("res_a"), dummyResource("res_b"), dummyResource("res_c")},
			resList)
	})
}

type (
	dummyResource  string
	dummyConstruct string
)

func (dr dummyResource) KlothoConstructRef() []core.AnnotationKey { return nil }

func (dr dummyResource) Id() core.ResourceId {
	return core.ResourceId{Provider: "test", Type: "", Name: string(dr)}
}

func (dc dummyConstruct) Provenance() core.AnnotationKey { return core.AnnotationKey{} }

func (dc dummyConstruct) Id() string { return string(dc) }
