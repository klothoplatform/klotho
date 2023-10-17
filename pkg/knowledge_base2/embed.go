package knowledgebase2

import (
	"errors"
	"fmt"
	"io/fs"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

func NewKBFromFs(resources, edges fs.FS) (*KnowledgeBase, error) {
	kb := NewKB()
	templates, err := TemplatesFromFs(resources)
	if err != nil {
		return nil, err
	}
	edgeTemplates, err := EdgeTemplatesFromFs(edges)
	if err != nil {
		return nil, err
	}

	var errs error
	for _, template := range templates {
		err = kb.AddResourceTemplate(template)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("error adding resource template %s: %w", template.QualifiedTypeName, err))
		}
	}
	for _, template := range edgeTemplates {
		err = kb.AddEdgeTemplate(template)
		if err != nil {
			errs = errors.Join(errs,
				fmt.Errorf("error adding edge template %s -> %s: %w",
					template.Source.QualifiedTypeName(),
					template.Target.QualifiedTypeName(),
					err),
			)
		}
	}
	return kb, errs
}

func TemplatesFromFs(dir fs.FS) (map[construct.ResourceId]*ResourceTemplate, error) {
	templates := map[construct.ResourceId]*ResourceTemplate{}
	err := fs.WalkDir(dir, ".", func(path string, d fs.DirEntry, nerr error) error {
		fmt.Println(d.Name())
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
		fmt.Println(d.Name())
		if d.IsDir() {
			return nil
		}
		f, err := dir.Open(path)
		if err != nil {
			zap.S().Errorf("Error opening edge template: %s", err)
			return errors.Join(nerr, err)
		}

		edgeTemplate := &EdgeTemplate{}
		err = yaml.NewDecoder(f).Decode(edgeTemplate)
		if err != nil {
			zap.S().Errorf("Error decoding edge template: %s", err)
			return errors.Join(nerr, err)
		}

		id := edgeTemplate.Source.QualifiedTypeName() + "->" + edgeTemplate.Target.QualifiedTypeName()
		if err != nil {
			zap.S().Errorf("Error unmarshalling edge template id: %s", err)
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
