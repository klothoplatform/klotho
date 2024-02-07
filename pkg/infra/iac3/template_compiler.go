package iac3

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/fs"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	kio "github.com/klothoplatform/klotho/pkg/io"
	"go.uber.org/zap"
)

type TemplatesCompiler struct {
	templates *templateStore

	graph construct.Graph
	vars  variables
}

// globalVariables are variables set in the global template and available to all resources
var globalVariables = map[string]struct{}{
	"kloConfig":  {},
	"awsConfig":  {},
	"protect":    {},
	"awsProfile": {},
	"accountId":  {},
	"region":     {},
	"aws":        {},
	"pulumi":     {},
}

type PackageJsonFile struct {
	Dependencies    map[string]string
	DevDependencies map[string]string
	OtherFields     map[string]json.RawMessage
}

func (tc TemplatesCompiler) PackageJSON() (*PackageJsonFile, error) {
	resources, err := construct.ReverseTopologicalSort(tc.graph)
	if err != nil {
		return nil, err
	}
	var errs error
	mainPJson := PackageJsonFile{}
	for _, id := range resources {
		pJson, err := tc.GetPackageJSON(id)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		if pJson != nil {
			mainPJson.Merge(pJson)
		}
	}
	return &mainPJson, errs
}

func (tc TemplatesCompiler) GetPackageJSON(v construct.ResourceId) (*PackageJsonFile, error) {
	templateFilePath := v.Provider + "/" + v.Type + `/package.json`
	contents, err := fs.ReadFile(tc.templates.fs, templateFilePath)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, nil

	case err != nil:
		return nil, err
	}
	var packageContent PackageJsonFile
	err = json.NewDecoder(bytes.NewReader(contents)).Decode(&packageContent)
	if err != nil {
		return &packageContent, err
	}
	return &packageContent, nil
}

func (f *PackageJsonFile) Merge(other *PackageJsonFile) {
	if f.Dependencies == nil {
		f.Dependencies = make(map[string]string)
	}
	for k, v := range other.Dependencies {
		currentVersion, ok := f.Dependencies[k]
		if ok {
			if currentVersion != v {
				zap.S().Warnf(`Found conflicting dependencies in package.json.
Found version of package, %s = %s.
Found version of package, %s = %s.
Using version %s`, k, currentVersion, k, v, currentVersion)
			}
		} else {
			f.Dependencies[k] = v
		}
	}

	if f.DevDependencies == nil {
		f.DevDependencies = make(map[string]string)
	}
	for k, v := range other.DevDependencies {
		f.DevDependencies[k] = v
	}

	// Ignore all other (non-supported / unmergeable) fields
}

func (f *PackageJsonFile) UnmarshalJSON(b []byte) error {
	var m map[string]json.RawMessage
	err := json.Unmarshal(b, &m)
	if err != nil {
		return err
	}

	if deps, ok := m["dependencies"]; ok {
		err = json.Unmarshal(deps, &f.Dependencies)
		if err != nil {
			return err
		}
		delete(m, "dependencies")
	}

	if deps, ok := m["devDependencies"]; ok {
		err = json.Unmarshal(deps, &f.DevDependencies)
		if err != nil {
			return err
		}
		delete(m, "devDependencies")
	}

	f.OtherFields = m

	return nil
}

func (f *PackageJsonFile) Path() string {
	return "package.json"
}

func (f *PackageJsonFile) WriteTo(w io.Writer) (n int64, err error) {
	m := map[string]interface{}{
		"dependencies":    f.Dependencies,
		"devDependencies": f.DevDependencies,
	}
	for k, v := range f.OtherFields {
		m[k] = v
	}
	h := &kio.CountingWriter{Delegate: w}
	enc := json.NewEncoder(h)
	enc.SetIndent("", "    ")
	err = enc.Encode(m)
	return int64(h.BytesWritten), err
}

func (f *PackageJsonFile) Clone() kio.File {
	clone := &PackageJsonFile{
		Dependencies:    make(map[string]string, len(f.Dependencies)),
		DevDependencies: make(map[string]string, len(f.DevDependencies)),
		OtherFields:     make(map[string]json.RawMessage, len(f.OtherFields)),
	}
	for k, v := range f.Dependencies {
		clone.Dependencies[k] = v
	}
	for k, v := range f.DevDependencies {
		clone.DevDependencies[k] = v
	}
	for k, v := range f.OtherFields {
		clone.OtherFields[k] = v
	}
	return clone
}
