package iac3

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/lang/javascript"
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
}

func (tc TemplatesCompiler) PackageJSON() (*javascript.NodePackageJson, error) {
	resources, err := construct.ReverseTopologicalSort(tc.graph)
	if err != nil {
		return nil, err
	}
	var errs error
	mainPJson := javascript.NodePackageJson{}
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

func (tc TemplatesCompiler) GetPackageJSON(v construct.ResourceId) (*javascript.NodePackageJson, error) {
	templateFilePath := v.Provider + "/" + v.Type + `/package.json`
	contents, err := fs.ReadFile(tc.templates.fs, templateFilePath)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, nil

	case err != nil:
		return nil, err
	}
	var packageContent javascript.NodePackageJson
	err = json.NewDecoder(bytes.NewReader(contents)).Decode(&packageContent)
	if err != nil {
		return &packageContent, err
	}
	return &packageContent, nil
}
