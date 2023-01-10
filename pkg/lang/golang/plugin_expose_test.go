package golang

import (
	"strings"
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

var testRestAPIHandler = &restAPIHandler{
	log:  zap.L(),
	Unit: &core.ExecutionUnit{Name: "testUnit"},
}

func Test_findHttpListenServe(t *testing.T) {
	tests := []struct {
		name         string
		source       string
		expectAppVar string
		expectErr    bool
	}{
		{
			name: "simple http listen and serve",
			source: `
				/* @klotho::expose {
				*   target = "public"
				*   id = "app"
				* }
				*/
				http.ListenAndServe(":3000", r)`,
			expectAppVar: "r",
		},
		{
			name: "incorrect http listen and serve",
			source: `
				/* @klotho::expose {
				*   target = "public"
				*   id = "app"
				* }
				*/
				http.TalkAndTake(":3000", r)`,
			expectAppVar: "",
			expectErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			f, err := core.NewSourceFile("", strings.NewReader(tt.source), language)
			if !assert.NoError(err) {
				return
			}
			var annot *core.Annotation
			for _, v := range f.Annotations() {
				annot = v
				break
			}
			listener, err := testRestAPIHandler.findHttpListenAndServe(annot, f)

			if tt.expectErr {
				assert.Error(err)
				return
			}

			if !assert.NotNil(listener.Expression, "error in test source app definition function") {
				return
			}

			assert.Equal(tt.expectAppVar, listener.Identifier.Content(f.Program()))
		})
	}
}

func Test_findChiRouterDefinition(t *testing.T) {
	tests := []struct {
		name         string
		source       string
		expectAppVar string
		expectErr    bool
	}{
		{
			name:         "simple chi router definition",
			source:       `r := chi.NewRouter()`,
			expectAppVar: "r",
			expectErr:    false,
		},
		// Right now we assume the var router found in the listen and serve is the same var with the router definition
		// In the future we may want to account for var reassignment
		{
			name:         "incorrect router name",
			source:       `r := chi.NewRouter()`,
			expectAppVar: "test",
			expectErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			f, err := core.NewSourceFile("", strings.NewReader(tt.source), language)
			if !assert.NoError(err) {
				return
			}
			router, err := testRestAPIHandler.findChiRouterDefinition(f, tt.expectAppVar)
			if tt.expectErr {
				assert.Error(err)
				return
			}

			assert.Equal(tt.expectAppVar, router.Identifier.Content(f.Program()))
		})
	}
}

func Test_findImports(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			name: "simple import block",
			source: `
			import (
				"fmt"
				"net/http"
					
				"github.com/go-chi/chi"
				"github.com/go-chi/chi/middleware"
			)`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			f, err := core.NewSourceFile("", strings.NewReader(tt.source), language)
			if !assert.NoError(err) {
				return
			}
			importsNode, _ := testRestAPIHandler.FindImports(f)

			assert.NotNil(importsNode)
		})
	}
}
