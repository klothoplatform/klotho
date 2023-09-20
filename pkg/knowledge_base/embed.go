package knowledgebase

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/klothoplatform/klotho/pkg/construct"
	"gopkg.in/yaml.v2"
)

func TemplatesFromFs(dir fs.FS) (map[construct.ResourceId]*ResourceTemplate, error) {
	templates := map[construct.ResourceId]*ResourceTemplate{}
	err := fs.WalkDir(dir, ".", func(path string, d fs.DirEntry, nerr error) error {
		if d.IsDir() {
			return nil
		}
		f, err := dir.Open(path)
		if err != nil {
			return errors.Join(nerr, err)
		}

		resTemplate := &ResourceTemplate{}
		err = yaml.NewDecoder(f).Decode(resTemplate)
		if err != nil {
			return errors.Join(nerr, err)
		}

		id := construct.ResourceId{Provider: resTemplate.Provider, Type: resTemplate.Type}
		if templates[id] != nil {
			return errors.Join(nerr, fmt.Errorf("duplicate template for %s", id))
		}
		templates[id] = resTemplate
		return nil
	})
	return templates, err
}
