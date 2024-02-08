package iac

import (
	"strings"

	construct "github.com/klothoplatform/klotho/pkg/construct"
)

type variables map[construct.ResourceId]string

var reservedVariables = map[string]struct{}{
	// This list from https://github.com/microsoft/TypeScript/issues/2536#issuecomment-87194347
	// typescript reserved keywords that cannot be variable names
	"break":      {},
	"case":       {},
	"catch":      {},
	"class":      {},
	"const":      {},
	"continue":   {},
	"debugger":   {},
	"default":    {},
	"delete":     {},
	"do":         {},
	"else":       {},
	"enum":       {},
	"export":     {},
	"extends":    {},
	"false":      {},
	"finally":    {},
	"for":        {},
	"function":   {},
	"if":         {},
	"import":     {},
	"in":         {},
	"instanceof": {},
	"new":        {},
	"null":       {},
	"return":     {},
	"super":      {},
	"switch":     {},
	"this":       {},
	"throw":      {},
	"true":       {},
	"try":        {},
	"typeof":     {},
	"var":        {},
	"void":       {},
	"while":      {},
	"with":       {},
	"as":         {},
	"implements": {},
	"interface":  {},
	"let":        {},
	"package":    {},
	"private":    {},
	"protected":  {},
	"public":     {},
	"static":     {},
	"yield":      {},
}

func VariablesFromGraph(g construct.Graph) (variables, error) {
	resources, err := construct.ReverseTopologicalSort(g)
	if err != nil {
		return nil, err
	}

	vars := make(variables, len(resources))

	type varInfo struct {
		all   []construct.ResourceId
		types map[string][]construct.ResourceId
	}

	nameInfo := make(map[string]*varInfo)
	for _, r := range resources {
		info, ok := nameInfo[r.Name]
		if !ok {
			info = &varInfo{
				types: make(map[string][]construct.ResourceId),
			}
			nameInfo[r.Name] = info
		}
		info.all = append(info.all, r)
		info.types[r.Type] = append(info.types[r.Type], r)
	}

	sanitizeName := func(parts ...string) string {
		for i, a := range parts {
			a = strings.ToLower(a)
			a = strings.ReplaceAll(a, "-", "_")
			parts[i] = a
		}
		return strings.Join(parts, "_")
	}

	for _, r := range resources {
		info := nameInfo[r.Name]
		// if there's only one resource wanting the name, it gets it
		if len(info.all) == 1 {
			_, isGlobal := globalVariables[r.Name]
			_, isReserved := reservedVariables[r.Name]
			if !isGlobal && !isReserved {
				vars[r] = sanitizeName(r.Name)
				continue
			}
		}

		typeResources := info.types[r.Type]

		// Type + Name unambiguously identifies the resource
		if len(typeResources) == 1 {
			vars[r] = sanitizeName(r.Type, r.Name)
			continue
		}

		if len(info.all) == len(typeResources) {
			// Namespace + Name unambiguously identifies the resource
			vars[r] = sanitizeName(r.Namespace, r.Name)
			continue
		}

		// This doesn't account for providers being different (and the rest being the same),
		// but the chances of that are low. So not implementing that until we have a real use case.

		vars[r] = sanitizeName(r.Type, r.Namespace, r.Name)
	}
	return vars, nil
}
