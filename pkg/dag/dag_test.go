package dag

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"testing"
)

func TestEmptyGraph(t *testing.T) {
	assert := assert.New(t)
	d := NewDag[DummyVertex]()
	assert.Empty(d.Roots())
}

func TestSimpleGraph(t *testing.T) {
	// A ┬─➤ B
	//   └─➤ C
	a, b, c := DummyVertex("a"), DummyVertex("b"), DummyVertex("c")
	d := NewDag[DummyVertex]()
	d.AddVertex(a)
	d.AddVertex(b)
	d.AddVertex(c)
	d.AddEdge(a, b)
	d.AddEdge(a, c)

	test(t, "roots", func(assert *assert.Assertions) {
		assert.Equal([]DummyVertex{a}, d.Roots())
	})
	test(t, "outgoing nodes", func(assert *assert.Assertions) {
		assert.ElementsMatch([]DummyVertex{b, c}, d.OutgoingVertices(a))
	})
	test(t, "outgoing edges", func(assert *assert.Assertions) {
		assert.ElementsMatch(
			[]Edge[DummyVertex]{
				Edge[DummyVertex]{
					Source:      a,
					Destination: b,
				},
				Edge[DummyVertex]{
					Source:      a,
					Destination: c,
				},
			},
			d.OutgoingEdges(a))
	})
}

func TestNegativeCases(t *testing.T) {
	test(t, "duplicate vertex", func(assert *assert.Assertions) {
		d := NewDag[DummyVertex]()
		v := DummyVertex("dummy")
		logged := captureLogs(func() {
			d.AddVertex(v)
			d.AddVertex(v)
		})
		assert.Equal([]DummyVertex{v}, d.Roots())
		assert.Equal(`Ignoring duplicate vertex "dummy"`, logged)
	})
	test(t, "duplicate edge", func(assert *assert.Assertions) {
		d := NewDag[DummyVertex]()
		v1 := DummyVertex("hello")
		v2 := DummyVertex("world")
		logged := captureLogs(func() {
			d.AddVertex(v1)
			d.AddVertex(v2)
			d.AddEdge(v1, v2)
			d.AddEdge(v1, v2)
		})
		assert.Equal(
			[]Edge[DummyVertex]{
				Edge[DummyVertex]{
					Source:      v1,
					Destination: v2,
				},
			},
			d.OutgoingEdges(v1))
		assert.Equal(`Ignoring duplicate edge from "hello" to "world"`, logged)
	})
	test(t, "edge to self", func(assert *assert.Assertions) {
		d := NewDag[DummyVertex]()
		v := DummyVertex("dummy")
		logged := captureLogs(func() {
			d.AddVertex(v)
			d.AddEdge(v, v)
		})
		assert.Equal([]DummyVertex{v}, d.Roots())
		assert.Equal(`Ignoring self-referential vertex "dummy"`, logged)
	})
	test(t, "cycle", func(assert *assert.Assertions) {
		d := NewDag[DummyVertex]()
		v1 := DummyVertex("hello")
		v2 := DummyVertex("world")
		logged := captureLogs(func() {
			d.AddVertex(v1)
			d.AddVertex(v2)
			d.AddEdge(v1, v2)
			d.AddEdge(v2, v1)
		})
		assert.Equal(
			[]Edge[DummyVertex]{
				Edge[DummyVertex]{
					Source:      v1,
					Destination: v2,
				},
			},
			d.OutgoingEdges(v1))
		assert.Equal(`Ignoring edge from "world" to "hello" because it would introduce a cyclic reference`, logged)
	})
}

type DummyVertex string

func (v DummyVertex) Id() string {
	return string(v)
}

func test(t *testing.T, name string, f func(assert *assert.Assertions)) {
	t.Run(name, func(t *testing.T) {
		assert := assert.New(t)
		f(assert)
	})
}

func captureLogs(f func()) string {
	buf := bytes.Buffer{}
	cfg := zap.NewDevelopmentConfig()
	cfg.OutputPaths = []string{}
	cfg.ErrorOutputPaths = []string{}
	capturingLogger, _ := cfg.Build(zap.Hooks(func(entry zapcore.Entry) error {
		if buf.Len() > 0 {
			buf.WriteRune('\n')
		}
		buf.WriteString(entry.Message)
		return nil
	}))
	restoreLogger := zap.ReplaceGlobals(capturingLogger)
	defer restoreLogger()

	f()

	if err := capturingLogger.Sync(); err != nil {
		buf.WriteString("ERROR: ")
		buf.WriteString(err.Error())
	}
	return buf.String()
}
