package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/stretchr/testify/assert"
)

func TestCreateElasticache(t *testing.T) {
	dag := core.NewResourceGraph()
	cfg := &config.Application{AppName: "test"}
	source := &core.RedisNode{}

	ec := CreateElasticache(cfg, dag, source)

	assert := assert.New(t)

	assert.NotNil(ec)
	assert.NotNil(dag.GetDependency(ec.Id(), ec.CloudwatchGroup.Id()))
	assert.NotNil(dag.GetDependency(ec.Id(), ec.SubnetGroup.Id()))
	for _, sn := range ec.SubnetGroup.Subnets {
		assert.NotNil(dag.GetDependency(ec.SubnetGroup.Id(), sn.Id()))
	}
	for _, sg := range ec.SecurityGroups {
		assert.NotNil(dag.GetDependency(ec.Id(), sg.Id()))
	}
	if assert.Len(ec.KlothoConstructRef(), 1) {
		assert.Equal(source.Provenance(), ec.KlothoConstructRef()[0])
	}
}
