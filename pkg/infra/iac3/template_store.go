package iac3

import (
	"fmt"
	"io/fs"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
)

type templateStore struct {
	fs                fs.FS
	resourceTemplates map[string]*ResourceTemplate
}

func (ts *templateStore) ResourceTemplate(id construct.ResourceId) (*ResourceTemplate, error) {
	typeName := id.QualifiedTypeName()
	if ts.resourceTemplates == nil {
		ts.resourceTemplates = make(map[string]*ResourceTemplate)
	}
	tmpl, ok := ts.resourceTemplates[typeName]
	if ok {
		return tmpl, nil
	}

	f, err := ts.fs.Open(id.Provider + "/" + id.Type + `/factory.ts`)
	if err != nil {
		return nil, fmt.Errorf("could not find template for %s: %w", typeName, err)
	}
	template, err := ParseTemplate(typeName, f)
	if err != nil {
		return nil, fmt.Errorf("could not parse template for %s: %w", typeName, err)
	}
	ts.resourceTemplates[typeName] = template
	return template, nil
}
