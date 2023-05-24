package coretesting

import (
	"reflect"
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/stretchr/testify/assert"
)

type CreateCase[P any, R core.ExpandableResource[P]] struct {
	Name     string
	Existing R
	Params   P
	Want     ResourcesExpectation
	Check    func(assertions *assert.Assertions, resource R)
	WantErr  bool
}

func (tt CreateCase[P, R]) Run(t *testing.T) {
	assert := assert.New(t)

	dag := core.NewResourceGraph()
	// We have to use reflect, since Go doesn't know that R is a pointer, or even something whose zero-type is
	// comparable (so we can't do "var R zero; if tt.Existing != zero").
	if !reflect.ValueOf(tt.Existing).IsZero() {
		dag.AddResource(tt.Existing)
	}

	var res R
	rType := reflect.TypeOf(tt.Existing)
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
