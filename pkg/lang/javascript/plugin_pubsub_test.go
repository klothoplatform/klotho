package javascript

import (
	"strings"
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func TestPubSub_rewriteFileEmitters(t *testing.T) {
	defer zap.ReplaceGlobals(zaptest.NewLogger(t))()

	tests := []struct {
		name    string
		source  string // source is attributed to a file named "test.js"
		varSpec VarSpec
		want    error
	}{
		{
			name: "simple test",
			source: `/* @klotho::pubsub {
				*  id = "myEmitter"
				* }
				*/
				exports.MyEmitter = new events.EventEmitter();`,
			varSpec: VarSpec{DefinedIn: "test.js", VarName: "MyEmitter"},
			want:    nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			p := &Pubsub{
				result: new(core.CompilationResult),
				deps:   new(core.Dependencies),
			}

			f, err := NewFile("test.js", strings.NewReader(tt.source))
			if !assert.NoError(err) {
				return
			}
			var annot *core.Annotation
			for _, v := range f.Annotations() {
				annot = v
				break
			}

			varDec := VarDeclarations{tt.varSpec: &VarParseStructs{Annotation: annot}}
			got := p.rewriteFileEmitters(f, varDec)
			assert.Equal(tt.want, got)
			assert.Contains(string(f.Program()), `exports.MyEmitter = new emitterRuntime.Emitter("test.js", "MyEmitter", "myEmitter");`)
		})
	}
}

func TestPubsub_findProducerTopics(t *testing.T) {
	defer zap.ReplaceGlobals(zaptest.NewLogger(t))()

	tests := []struct {
		name   string
		source string // source is attributed to a file named "test.js"
		spec   VarSpec
		want   []string
	}{
		{
			name:   "no topics",
			source: "const a = 2",
			spec:   VarSpec{DefinedIn: "test.js", InternalName: "ev", VarName: "emitter"},
			want:   nil,
		},
		{
			name:   "one topics",
			source: "ev.emit('a')",
			spec:   VarSpec{DefinedIn: "test.js", InternalName: "ev", VarName: "emitter"},
			want:   []string{"a"},
		},
		{
			name:   "multiple topics",
			source: "ev.emit('a'); ev.emit('b')",
			spec:   VarSpec{DefinedIn: "test.js", InternalName: "ev", VarName: "emitter"},
			want:   []string{"a", "b"},
		},
		{
			name:   "not emit",
			source: "ev.on('a')",
			spec:   VarSpec{DefinedIn: "test.js", InternalName: "ev", VarName: "emitter"},
			want:   nil,
		},
		{
			name:   "wrong var",
			source: "ev.emit('a')",
			spec:   VarSpec{DefinedIn: "test.js", InternalName: "somethingelse", VarName: "emitter"},
			want:   nil,
		},
		{
			name: "imported uses exportname",
			source: `const e = require('./otherfile')
e.emitter.emit('a')`,
			spec: VarSpec{DefinedIn: "otherfile.js", InternalName: "ev", VarName: "emitter"},
			want: []string{"a"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			p := &Pubsub{
				result: new(core.CompilationResult),
				deps:   new(core.Dependencies),
			}

			f, err := NewFile("test.js", strings.NewReader(tt.source))
			if !assert.NoError(err) {
				return
			}

			got := p.findPublisherTopics(f, tt.spec)
			assert.Equal(tt.want, got)
		})
	}
}

func TestPubsub_findSubscriberTopics(t *testing.T) {
	defer zap.ReplaceGlobals(zaptest.NewLogger(t))()

	tests := []struct {
		name   string
		source string // source is attributed to a file named "test.js"
		spec   VarSpec
		want   []string
	}{
		{
			name:   "no topics",
			source: "const a = 2",
			spec:   VarSpec{DefinedIn: "test.js", InternalName: "ev", VarName: "emitter"},
			want:   nil,
		},
		{
			name:   "one topics",
			source: "ev.on('a', () => {})",
			spec:   VarSpec{DefinedIn: "test.js", InternalName: "ev", VarName: "emitter"},
			want:   []string{"a"},
		},
		{
			name:   "multiple topics",
			source: "ev.on('a', () => {}); ev.on('b', () => {})",
			spec:   VarSpec{DefinedIn: "test.js", InternalName: "ev", VarName: "emitter"},
			want:   []string{"a", "b"},
		},
		{
			name:   "not emit",
			source: "ev.emit('a')",
			spec:   VarSpec{DefinedIn: "test.js", InternalName: "ev", VarName: "emitter"},
			want:   nil,
		},
		{
			name:   "wrong var",
			source: "ev.on('a', () => {})",
			spec:   VarSpec{DefinedIn: "test.js", InternalName: "somethingelse", VarName: "emitter"},
			want:   nil,
		},
		{
			name: "imported uses exportname",
			source: `const e = require('./otherfile')
e.emitter.on('a', () => {})`,
			spec: VarSpec{DefinedIn: "otherfile.js", InternalName: "ev", VarName: "emitter"},
			want: []string{"a"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			p := &Pubsub{
				result: new(core.CompilationResult),
				deps:   new(core.Dependencies),
			}

			f, err := NewFile("test.js", strings.NewReader(tt.source))
			if !assert.NoError(err) {
				return
			}

			got := p.findSubscriberTopics(f, tt.spec)
			assert.Equal(tt.want, got)
		})
	}
}
