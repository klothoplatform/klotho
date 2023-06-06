package core

import (
	"encoding/json"
	"fmt"
)

type (
	// FileDependencies is a map from file (by path) to the other files it imports
	FileDependencies map[string]Imported

	// Imported is a map from the imported file to the set of references used by the original file
	Imported map[string]References

	// References is a set of references that are being used from the file
	References map[string]struct{}
)

func (deps FileDependencies) Add(other FileDependencies) {
	for k, v := range other {
		imports, alreadyThere := deps[k]
		if !alreadyThere {
			imports = make(Imported)
			deps[k] = imports
		}
		imports.AddAll(v)
	}
}

func (i Imported) AddAll(other Imported) {
	for k, v := range other {
		refs, alreadyThere := i[k]
		if !alreadyThere {
			refs = make(References)
			i[k] = refs
		}
		refs.AddAll(v)
	}
}

func (r References) Add(ref string) {
	r[ref] = struct{}{}
}

func (r References) AddAll(other References) {
	for k := range other {
		r.Add(k)
	}
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
