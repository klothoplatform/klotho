package javascript

import (
	"strings"
	"testing"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func TestVarfinder_DiscoverDeclarations(t *testing.T) {
	defer zap.ReplaceGlobals(zaptest.NewLogger(t))()

	tests := []struct {
		name    string
		sources map[string]string
		want    []VarSpec
	}{
		{
			name: "no emitters",
			sources: map[string]string{
				"test.js": "const a = 2",
			},
			want: []VarSpec{},
		},
		{
			name: "non-exported emitter",
			sources: map[string]string{
				"test.js": `// @klotho::pubsub
const e = new EventEmitter()`,
			},
			want: []VarSpec{},
		},
		{
			name: "single emitter",
			sources: map[string]string{
				"test.js": `// @klotho::pubsub
exports.e = new EventEmitter()`,
			},
			want: []VarSpec{
				{DefinedIn: "test.js", InternalName: "exports.e", VarName: "e"},
			},
		},
		{
			name: "export renamed",
			sources: map[string]string{
				"test.js": `// @klotho::pubsub
const e = new EventEmitter()
exports.a = e`,
			},
			want: []VarSpec{
				{DefinedIn: "test.js", InternalName: "e", VarName: "a"},
			},
		},
		{
			name: "two emitters one file",
			sources: map[string]string{
				"test.js": `// @klotho::pubsub { id = "e" }
exports.e = new EventEmitter()

// @klotho::pubsub { id = "e2" }
exports.e2 = new EventEmitter()`,
			},
			want: []VarSpec{
				{DefinedIn: "test.js", InternalName: "exports.e", VarName: "e"},
				{DefinedIn: "test.js", InternalName: "exports.e2", VarName: "e2"},
			},
		},
		{
			name: "multiple file emitters",
			sources: map[string]string{
				"test.js": `// @klotho::pubsub
exports.e1 = new EventEmitter()`,
				"test2.js": `// @klotho::pubsub
exports.e2 = new EventEmitter()`,
			},
			want: []VarSpec{
				{DefinedIn: "test.js", InternalName: "exports.e1", VarName: "e1"},
				{DefinedIn: "test2.js", InternalName: "exports.e2", VarName: "e2"},
			},
		},
		{
			name: "qualified instantiation of non-events EventEmitter",
			sources: map[string]string{
				"test.js": `// @klotho::pubsub
exports.e1 = new something.EventEmitter()`,
			},
			want: []VarSpec{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			files := make(map[string]core.File)
			for filename, content := range tt.sources {
				f, err := NewFile(filename, strings.NewReader(content))
				if !assert.NoError(err) {
					return
				}
				files[f.Path()] = f
			}

			// There's nothing special about pubsub here; we just picked it as the canonical test type for historical reasons.
			vars := DiscoverDeclarations(files, pubsubVarType, pubsubVarTypeModule, true, FilterByCapability(annotation.PubSubCapability))

			for _, want := range tt.want {
				assert.Contains(vars, want)
			}
			assert.Len(vars, len(tt.want))
		})
	}
}

func TestVarFinder_parseNode(t *testing.T) {
	defer zap.ReplaceGlobals(zaptest.NewLogger(t))()

	tests := []struct {
		name         string
		source       string // source is attributed to a file named "test.js"
		wantInternal string
		wantExport   string
		wantErr      bool
	}{
		{
			name: "simple export assign",
			source: `// @klotho::pubsub
exports.e = new EventEmitter();`,
			wantInternal: "exports.e",
			wantExport:   "e",
			wantErr:      false,
		},
		{
			name: "simple re-export",
			source: `// @klotho::pubsub
const e = new EventEmitter();
exports.e2 = e`,
			wantInternal: "e",
			wantExport:   "e2",
			wantErr:      false,
		},
		{
			name: "not an emitter",
			source: `// @klotho::pubsub
exports.e = new Map()`,
			wantErr: true,
		},
		{
			name: "qualified events instantiation",
			source: `// @klotho::pubsub
exports.e1 = new events.EventEmitter()`,
			wantInternal: "exports.e1",
			wantExport:   "e1",
			wantErr:      false,
		},
		{
			name: "qualified instantiation of non-events EventEmitter",
			source: `// @klotho::pubsub
exports.e1 = new something.EventEmitter()`,
			wantInternal: "",
			wantExport:   "",
			wantErr:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			f, err := NewFile("test.js", strings.NewReader(tt.source))
			if !assert.NoError(err) {
				return
			}

			var annots []*core.Annotation
			for _, a := range f.Annotations() {
				if a.Capability.Name == core.PubSubKind {
					annots = append(annots, a)
				}
			}
			if !assert.Len(annots, 1) {
				return
			}
			annot := annots[0]

			// There's nothing special about pubsub here; we just picked it as the canonical test type for historical reasons.
			vf := varFinder{
				queryMatchType:       pubsubVarType,
				queryMatchTypeModule: pubsubVarTypeModule,
			}
			gotInternal, gotExport, err := vf.parseNode(f, annot)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.wantInternal, gotInternal)
			assert.Equal(tt.wantExport, gotExport)
		})
	}
}
