package javascript

import (
	"encoding/json"
	"io"
	"sync"

	"github.com/klothoplatform/klotho/pkg/core"
)

type PackageFile struct {
	path    string
	Content *NodePackageJson
}

func NewPackageFile(path string, content io.Reader) (*PackageFile, error) {
	f := &PackageFile{
		path: path,
	}

	err := json.NewDecoder(content).Decode(&f.Content)
	if err != nil {
		err = core.WrapErrf(err, "could not decode json for %s", path)
	}
	return f, err
}

func (f *PackageFile) Clone() core.File {
	nf := &PackageFile{
		path:    f.path,
		Content: f.Content.Clone(),
	}
	return nf
}

func (f *PackageFile) Path() string {
	return f.path
}

func (f *PackageFile) WriteTo(out io.Writer) (int64, error) {
	counter := &core.CountingWriter{Delegate: out}
	enc := json.NewEncoder(counter)
	enc.SetIndent("", "  ")
	err := enc.Encode(f.Content)
	if err != nil {
		return int64(counter.BytesWritten), err
	}
	return int64(counter.BytesWritten), nil
}

// NodePackageJson represents the type described in https://docs.npmjs.com/cli/v8/configuring-npm/package-json
type NodePackageJson struct {
	Dependencies    map[string]string
	DevDependencies map[string]string

	OtherFields map[string]json.RawMessage

	mu sync.Mutex
}

func (n *NodePackageJson) Clone() *NodePackageJson {
	n.mu.Lock()
	defer n.mu.Unlock()

	c := &NodePackageJson{
		Dependencies:    make(map[string]string),
		DevDependencies: make(map[string]string),
		OtherFields:     make(map[string]json.RawMessage),
	}
	for k, v := range n.Dependencies {
		c.Dependencies[k] = v
	}
	for k, v := range n.DevDependencies {
		c.DevDependencies[k] = v
	}
	for k, v := range n.OtherFields {
		c.OtherFields[k] = make(json.RawMessage, len(v))
		copy(c.OtherFields[k], v)
	}

	return c
}

func (n *NodePackageJson) Merge(other *NodePackageJson) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.Dependencies == nil {
		n.Dependencies = make(map[string]string)
	}
	for k, v := range other.Dependencies {
		n.Dependencies[k] = v
	}

	if n.DevDependencies == nil {
		n.DevDependencies = make(map[string]string)
	}
	for k, v := range other.DevDependencies {
		n.DevDependencies[k] = v
	}

	// Ignore all other (non-supported / unmergeable) fields
}

func (n *NodePackageJson) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"dependencies":    n.Dependencies,
		"devDependencies": n.DevDependencies,
	}
	for k, v := range n.OtherFields {
		m[k] = v
	}
	return json.Marshal(m)
}

func (n *NodePackageJson) UnmarshalJSON(b []byte) error {
	var m map[string]json.RawMessage
	err := json.Unmarshal(b, &m)
	if err != nil {
		return err
	}

	if deps, ok := m["dependencies"]; ok {
		err = json.Unmarshal(deps, &n.Dependencies)
		if err != nil {
			return err
		}
		delete(m, "dependencies")
	}

	if deps, ok := m["devDependencies"]; ok {
		err = json.Unmarshal(deps, &n.DevDependencies)
		if err != nil {
			return err
		}
		delete(m, "devDependencies")
	}

	n.OtherFields = m

	return nil
}
