package core

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/pelletier/go-toml/v2"
	sitter "github.com/smacker/go-tree-sitter"
)

type (
	Annotation struct {
		Capability *annotation.Capability
		// Node is the node that has been annotated; not the comment node representing the annotation itself.
		Node *sitter.Node
	}

	AnnotationKey struct {
		Capability string
		ID         string
	}

	AnnotationMap map[AnnotationKey]*Annotation
)

var (
	lineIndentRE = regexp.MustCompile(`(?m)^`)
)

func (a *Annotation) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"capability": a.Capability,
		"node": map[string]interface{}{
			"types": a.Node.String(),
		},
	}

	return json.Marshal(m)
}

func (a Annotation) Format(s fmt.State, verb rune) {
	fmt.Fprintf(s, "@klotho::%s", a.Capability.Name)
	if len(a.Capability.Directives) > 0 {
		if s.Flag('+') || s.Flag('#') {
			fmt.Fprintf(s, " {")

			dvals, _ := toml.Marshal(a.Capability.Directives)
			directives := strings.TrimRight(string(dvals), "\n") // remove trailing newline so it doesn't get indented
			directives = lineIndentRE.ReplaceAllString(directives, "     ")
			fmt.Fprintf(s, "\n%s\n}", directives)
		} else {
			fmt.Fprintf(s, " (%d directives)", len(a.Capability.Directives))
		}
	}
}

func (a Annotation) Key() AnnotationKey {
	return AnnotationKey{Capability: a.Capability.Name, ID: a.Capability.ID}
}

func (m AnnotationMap) Update(other AnnotationMap) {
	for k, v := range other {
		if ex, ok := m[k]; ok {
			// Update the contents not the pointer so existing annotation pointers are still valid
			*ex = *v
		} else {
			m[k] = v
		}
	}
	for k := range m {
		if _, ok := other[k]; !ok {
			delete(m, k)
		}
	}
}

func (m AnnotationMap) Add(a *Annotation) {
	m[a.Key()] = a
}
