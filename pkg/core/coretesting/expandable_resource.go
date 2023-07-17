package coretesting

import (
	"reflect"
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/engine/classification"
	"github.com/stretchr/testify/assert"
)

type CreateCase[P any, R core.ExpandableResource[P]] struct {
	Name     string
	Existing core.Resource
	Params   P
	Want     ResourcesExpectation
	Check    func(assertions *assert.Assertions, resource R)
	WantErr  bool
}

func (tt CreateCase[P, R]) Run(t *testing.T) {
	assert := assert.New(t)

	dag := core.NewResourceGraph()
	if tt.Existing != nil {
		dag.AddResource(tt.Existing)
	}

	var res R
	rType := reflect.TypeOf(res)
	if rType.Kind() == reflect.Pointer {
		res = reflect.New(rType.Elem()).Interface().(R)
	}
	err := res.Create(dag, tt.Params)
	if tt.WantErr {
		assert.Error(err)
		return
	} else if !assert.NoError(err) {
		return
	}
	tt.Want.Assert(t, dag)
	found := dag.GetResource(res.Id())
	if !assert.NotNil(found) {
		return
	}
	foundR := found.(R)

	if !assert.NotNil(tt.Check, "bug in test itself: no Case.Check provided!") {
		return
	}
	tt.Check(assert, foundR)
}

type ConfigureCase[P any, R core.ConfigurableResource[P]] struct {
	Name    string
	Params  P
	Want    R
	WantErr bool
}

func (tt ConfigureCase[P, R]) Run(t *testing.T) {
	assert := assert.New(t)

	var res R
	rType := reflect.TypeOf(res)
	if rType.Kind() == reflect.Pointer {
		res = reflect.New(rType.Elem()).Interface().(R)
	}
	err := res.Configure(tt.Params)
	if tt.WantErr {
		assert.Error(err)
		return
	} else if !assert.NoError(err) {
		return
	}
	assert.Equal(tt.Want, res)
}

type (
	OperationalResource interface {
		core.Resource
		MakeOperational(dag *core.ResourceGraph, appName string, classifier classification.Classifier) error
	}
)

type MakeOperationalCase[R OperationalResource] struct {
	Name                 string
	Resource             R
	Existing             []core.Resource
	ExistingDependencies []StringDep
	AppName              string
	Want                 ResourcesExpectation
	Check                func(assertions *assert.Assertions, resource R)
	WantErr              bool
}

func (tt MakeOperationalCase[R]) Run(t *testing.T) {
	assert := assert.New(t)

	dag := core.NewResourceGraph()
	dag.AddResource(tt.Resource)
	for _, res := range tt.Existing {
		dag.AddResource(res)
	}
	for _, dep := range tt.ExistingDependencies {
		dag.AddDependencyByString(dep.Source, dep.Destination, nil)
	}

	err := tt.Resource.MakeOperational(dag, tt.AppName, nil)
	if tt.WantErr {
		assert.Error(err)
		return
	} else if !assert.NoError(err) {
		return
	}
	tt.Want.Assert(t, dag)
	found := dag.GetResource(tt.Resource.Id())
	if !assert.NotNil(found) {
		return
	}
	foundR := found.(R)

	if !assert.NotNil(tt.Check, "bug in test itself: no Case.Check provided!") {
		return
	}
	tt.Check(assert, foundR)
}
