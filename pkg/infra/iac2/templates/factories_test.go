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
	topLevelAssert := assert.New(t)
	// Find all of the versions across all jsons
	versionsByName := make(map[string]map[string]struct{})
	err := fs.WalkDir(packageJsonsFs, ".", func(path string, d fs.DirEntry, err error) error {
		if !topLevelAssert.NoError(err) {
			return err
		}
		if d.IsDir() {
			return nil
		}
		contents, err := fs.ReadFile(packageJsonsFs, path)
		if !topLevelAssert.NoError(err) {
			return err
		}
		var jsonObj packageJson
		if err = json.Unmarshal(contents, &jsonObj); !topLevelAssert.NoError(err) {
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
	if !topLevelAssert.NoError(err) {
		return
	}
	topLevelAssert.NotEmpty(versionsByName)

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
