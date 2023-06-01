package templates

import (
	"embed"
	"encoding/json"
	"io/fs"
	"testing"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
	"github.com/stretchr/testify/assert"
)

//go:embed **/package.json
var packageJsonsFs embed.FS

func Test_SameVersionOfDeps(t *testing.T) {
	// Find all of the versions across all jsons
	versionsByName := make(map[string]map[string]struct{})
	fs.WalkDir(packageJsonsFs, ".", func(path string, d fs.DirEntry, err error) error {
		assert := assert.New(t)
		if !assert.NoError(err) {
			return err
		}
		if d.IsDir() {
			return nil
		}
		contents, err := fs.ReadFile(packageJsonsFs, path)
		if !assert.NoError(err) {
			return err
		}
		var jsonObj packageJson
		if err = json.Unmarshal(contents, &jsonObj); !assert.NoError(err) {
			return err
		}
		for name, version := range jsonObj.Dependencies {
			existing := versionsByName[name]
			if existing == nil {
				existing = make(map[string]struct{})
				versionsByName[name] = existing
			}
			existing[version] = struct{}{}
		}
		return nil
	})
	assert.New(t).NotEmpty(versionsByName)

	for name, versions := range versionsByName {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			assert.Equal(1, len(versions), `found multiple versions: %v`, collectionutil.Keys(versions))
		})

	}
}

type packageJson struct {
	Dependencies map[string]string `json:"dependencies"`
}
