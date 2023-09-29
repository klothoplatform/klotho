package knowledgebase2

import (
	"errors"
	"fmt"
	"io/fs"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"gopkg.in/yaml.v3"
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

		id := construct.ResourceId{}
		err = id.UnmarshalText([]byte(resTemplate.QualifiedTypeName))
		if err != nil {
			return errors.Join(nerr, err)
		}
		if templates[id] != nil {
			return errors.Join(nerr, fmt.Errorf("duplicate template for %s", id))
		}
		templates[id] = resTemplate
		return nil
	})
	return templates, err
}

func EdgeTemplatesFromFs(dir fs.FS) (map[string]*EdgeTemplate, error) {
	templates := map[string]*EdgeTemplate{}
	err := fs.WalkDir(dir, ".", func(path string, d fs.DirEntry, nerr error) error {
		if d.IsDir() {
			return nil
		}
		f, err := dir.Open(path)
		if err != nil {
			return errors.Join(nerr, err)
		}

		edgeTemplate := &EdgeTemplate{}
		err = yaml.NewDecoder(f).Decode(edgeTemplate)
		if err != nil {
			return errors.Join(nerr, err)
		}

		id := edgeTemplate.Source.QualifiedTypeName() + "->" + edgeTemplate.Target.QualifiedTypeName()
		if err != nil {
			return errors.Join(nerr, err)
		}
		if templates[id] != nil {
			return errors.Join(nerr, fmt.Errorf("duplicate template for %s", id))
		}
		templates[id] = edgeTemplate
		return nil
	})
	return templates, err
}