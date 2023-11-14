package knowledgebase2

import (
	"errors"
	"fmt"
	"io/fs"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

func NewKBFromFs(resources, edges, models fs.FS) (*KnowledgeBase, error) {
	kb := NewKB()
	templates, err := TemplatesFromFs(resources, models)
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

func ModelsFromFS(dir fs.FS) (map[string]*Model, error) {
	models := map[string]*Model{}
	err := fs.WalkDir(dir, ".", func(path string, d fs.DirEntry, nerr error) error {
		zap.S().Debug("Loading model: ", path)
		if d.IsDir() {
			return nil
		}
		f, err := dir.Open(path)
		if err != nil {
			return errors.Join(nerr, fmt.Errorf("error opening model file %s: %w", path, err))
		}

		model := &Model{}
		err = yaml.NewDecoder(f).Decode(model)
		if err != nil {
			return errors.Join(nerr, fmt.Errorf("error decoding model file %s: %w", path, err))
		}
		models[model.Name] = model
		return nil
	})

	for _, model := range models {
		uerr := updateModels(nil, model.Properties, models)
		if uerr != nil {
			err = errors.Join(err, uerr)
		}
	}
	return models, err
}

func TemplatesFromFs(dir, modelDir fs.FS) (map[construct.ResourceId]*ResourceTemplate, error) {
	templates := map[construct.ResourceId]*ResourceTemplate{}
	models, err := ModelsFromFS(modelDir)
	if err != nil {
		return nil, err
	}
	terr := fs.WalkDir(dir, ".", func(path string, d fs.DirEntry, nerr error) error {
		zap.S().Debug("Loading resource template: ", path)
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
			return errors.Join(nerr, fmt.Errorf("error decoding resource template %s: %w", path, err))
		}
		err = updateModels(nil, resTemplate.Properties, models)
		if err != nil {
			return errors.Join(nerr, fmt.Errorf("error updating models for resource template %s: %w", path, err))
		}
		id := construct.ResourceId{}
		err = id.UnmarshalText([]byte(resTemplate.QualifiedTypeName))
		if err != nil {
			return errors.Join(nerr, fmt.Errorf("error unmarshalling resource template id for %s: %w", path, err))
		}
		if templates[id] != nil {
			return errors.Join(nerr, fmt.Errorf("duplicate template for %s in %s", id, path))
		}
		templates[id] = resTemplate
		return nil
	})
	err = errors.Join(err, terr)
	return templates, err
}

func EdgeTemplatesFromFs(dir fs.FS) (map[string]*EdgeTemplate, error) {
	templates := map[string]*EdgeTemplate{}
	err := fs.WalkDir(dir, ".", func(path string, d fs.DirEntry, nerr error) error {
		zap.S().Debug("Loading edge template: ", path)
		if d.IsDir() {
			return nil
		}
		f, err := dir.Open(path)
		if err != nil {
			return errors.Join(nerr, fmt.Errorf("error opening edge template %s: %w", path, err))
		}

		edgeTemplate := &EdgeTemplate{}
		err = yaml.NewDecoder(f).Decode(edgeTemplate)
		if err != nil {
			return errors.Join(nerr, fmt.Errorf("error decoding edge template %s: %w", path, err))
		}
		if edgeTemplate.Source.IsZero() || edgeTemplate.Target.IsZero() {
			f, err := dir.Open(path)
			if err != nil {
				return errors.Join(nerr, fmt.Errorf("error opening edge template %s: %w", path, err))
			}
			multiEdgeTemplate := &MultiEdgeTemplate{}
			err = yaml.NewDecoder(f).Decode(multiEdgeTemplate)
			if err != nil {
				return errors.Join(nerr, fmt.Errorf("error decoding edge template %s: %w", path, err))
			}
			if !multiEdgeTemplate.Resource.IsZero() && (len(multiEdgeTemplate.Sources) > 0 || len(multiEdgeTemplate.Targets) > 0) {
				edgeTemplates := EdgeTemplatesFromMulti(*multiEdgeTemplate)
				for _, edgeTemplate := range edgeTemplates {
					id := edgeTemplate.Source.QualifiedTypeName() + "->" + edgeTemplate.Target.QualifiedTypeName()
					if templates[id] != nil {
						return errors.Join(nerr, fmt.Errorf("duplicate template for %s in %s", id, path))
					}
					et := edgeTemplate
					templates[id] = &et
				}
				return nil
			}
		}

		id := edgeTemplate.Source.QualifiedTypeName() + "->" + edgeTemplate.Target.QualifiedTypeName()
		if templates[id] != nil {
			return errors.Join(nerr, fmt.Errorf("duplicate template for %s in %s", id, path))
		}
		templates[id] = edgeTemplate
		return nil
	})
	return templates, err
}
