package execunit

import (
	"encoding/json"
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"go.uber.org/zap"
)

type (
	// FileDependencies is a map from file (by path) to the other files it imports
	FileDependencies map[string]Imported

	// Imported is a map from the imported file to the set of references used by the original file
	Imported map[string]References

	// References is a set of references that are being used from the file
	References map[string]struct{}
)

const (
	FileDependenciesResourceKind = "input_file_dependencies"
)

func (FileDependencies) Type() string { return "" }

// Key implements core.CloudResource
func (deps FileDependencies) Key() core.ResourceKey {
	return core.ResourceKey{
		Kind: FileDependenciesResourceKind,
	}
}

func (deps FileDependencies) Add(other FileDependencies) {
	for k, v := range other {
		if _, alreadyThere := deps[k]; alreadyThere {
			// This shouldn't happen, as long as each plugin sticks to its own files (ie python only does .py files,
			// etc)
			zap.S().Warnf("Multiple file dependencies found for %v. Will use one at random.", k)
		} else {
			deps[k] = v
		}
	}
}

func (r References) Add(ref string) {
	r[ref] = struct{}{}
}

func (r References) Clone() References {
	n := make(References)
	for k := range r {
		n[k] = struct{}{}
	}
	return n
}

func (r References) String() string {
	s := make([]string, 0, len(r))
	for k := range r {
		s = append(s, k)
	}
	return fmt.Sprintf("%v", s)
}

func (r References) MarshalJSON() ([]byte, error) {
	refs := make([]string, 0, len(r))
	for ref := range r {
		refs = append(refs, ref)
	}
	return json.Marshal(refs)
}
