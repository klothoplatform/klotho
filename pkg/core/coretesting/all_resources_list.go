package coretesting

import (
	"fmt"
	"go/types"
	"reflect"
	"runtime"
	"strings"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"golang.org/x/tools/go/packages"
)

type (
	Methods struct {
		// signatures is the set of all methods declared on a type. Each signature follows the general format:
		//
		//	<name> func(<args>) <return type>
		//
		// The args do not include the receiver type. For example:
		//
		//	KlothoConstructRef func() []github.com/klothoplatform/klotho/pkg/core.AnnotationKey
		signatures  map[string]struct{}
		isInterface bool
	}

	TypeRef struct {
		Pkg  string
		Name string
	}
)

// FindAllResources returns the TypeRefs for each of the resources you pass in, as well as for any other resources in
// the same-or-descendent packages from those resources, and also any resources in the caller's package (or
// descendents).
//
// For example, let's say you had:
//
//	└─ pkg/core/resources/
//	   ├─ aws/
//	   │  ├╴ResourceA
//	   │  ├╴ResourceB
//	   │  └─ subpackage/
//	   │     └─ ResourceC
//	   └─ gcp/
//	      └─ ResourceD
//
// If you invoked FindAllResources with { &ResourceA{} }, this would return TypeRefs for:
//   - ResourceA (because it was in the input)
//   - ResourceB (because it's in the same package as A)
//   - ResourceC (because it was in a subpackage of A)
//   - but *not* for ResourceD
//
// If there are any problems, this function will return nil, but also trigger a failure on the provided
// [assert.Assertions].
func FindAllResources(a *assert.Assertions, from []core.Resource) []TypeRef {
	// Find caller's package
	pc, _, _, ok := runtime.Caller(1)
	if !a.Truef(ok, "couldn't get caller info") {
		return nil
	}
	// callerFunc is something like "github.com/klothoplatform/klotho/pkg/infra/iac2.TestKnownTemplates.func2"
	// we want to extract "github.com/klothoplatform/klotho/pkg/infra/iac2"
	callerFunc := runtime.FuncForPC(pc).Name()
	callerFuncLastSlash := strings.LastIndexByte(callerFunc, '/')
	callerFuncDotAfterLastSlash := strings.IndexByte(callerFunc[callerFuncLastSlash:], '.') + callerFuncLastSlash
	callerPkg := callerFunc[:callerFuncDotAfterLastSlash]

	// Find the methods for core.Resource
	var t2 reflect.Type = reflect.TypeOf((*core.Resource)(nil)).Elem()
	coreResourceRef := TypeRef{
		Pkg:  t2.PkgPath(),
		Name: t2.Name(),
	}
	coreTypes, err := getTypesInPackage(coreResourceRef.Pkg)
	if !a.NoError(err) {
		return nil
	}
	coreResourceType := coreTypes[coreResourceRef]
	if !a.NotEmptyf(coreResourceType, `couldn't find %v!`, coreResourceRef) {
		return nil
	}

	// Find all packages in our "from" list"
	packageNames := make(map[string]struct{})
	packageNames[callerPkg] = struct{}{}
	for _, res := range from {
		resType := reflect.TypeOf(res)
		for resType.Kind() == reflect.Pointer {
			resType = resType.Elem()
		}
		packageNames[resType.PkgPath()] = struct{}{}
	}

	// Within those packages, find all structs that implement core.Resource

	// allTypesRef is basically a cache: for each TypeRef we see, is it a core.Resource?
	// We do this because we look for packages using `/...`, so it's very possible that we check a package and then its
	// sibling; this approach lets us avoid dupes. The concern isn't so much unnecessary compute, but that we don't want
	// to return dupes to the caller.
	allTypeRefs := make(map[TypeRef]bool)
	for pkg := range packageNames {
		resourcesTypes, err := getTypesInPackage(pkg + `/...`)
		if !a.NoError(err) {
			return nil
		}
		for ref, methods := range resourcesTypes {
			if _, alreadyThere := allTypeRefs[ref]; alreadyThere {
				continue
			}
			// Ignore all interfaces, and all structs/typedefs that don't implement core.Resource
			allTypeRefs[ref] = (!methods.isInterface) && methods.containsAllMethodsIn(coreResourceType)
		}
	}

	results := make([]TypeRef, 0, len(allTypeRefs))
	for ref, isResource := range allTypeRefs {
		if isResource {
			results = append(results, ref)
		}
	}
	return results
}

// getTypesInPackage finds all types within a package (which may be "..."-ed).
func getTypesInPackage(packageName string) (map[TypeRef]Methods, error) {
	config := &packages.Config{Mode: packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo}
	pkgs, err := packages.Load(config, packageName)
	if err != nil {
		return nil, err
	}
	result := make(map[TypeRef]Methods)
	for _, pkg := range pkgs {
		for _, obj := range pkg.TypesInfo.Defs {
			if obj == nil {
				continue
			}
			if _, ok := obj.(*types.TypeName); !ok {
				continue
			}
			typ, ok := obj.Type().(*types.Named)
			if !ok {
				continue
			}
			key := TypeRef{
				Pkg:  pkg.PkgPath,
				Name: obj.Name(),
			}
			result[key] = getMethods(typ)
		}
	}
	if len(result) == 0 {
		return nil, errors.Errorf(`couldn't find any packages in %s`, packageName)
	}
	return result, nil
}

func getMethods(t *types.Named) Methods {
	type hasMethods interface {
		NumMethods() int
		Method(int) *types.Func
	}
	result := Methods{}
	var tMethods hasMethods = t
	if underlyingInterface, ok := t.Underlying().(*types.Interface); ok {
		tMethods = underlyingInterface
		result.isInterface = true
	}
	result.signatures = make(map[string]struct{}, tMethods.NumMethods())
	for i := 0; i < tMethods.NumMethods(); i++ {
		method := tMethods.Method(i)
		result.signatures[fmt.Sprintf(`%s %s`, method.Name(), method.Type().String())] = struct{}{}
	}
	return result
}

func (m Methods) containsAllMethodsIn(other Methods) bool {
	for sig := range other.signatures {
		if _, exists := m.signatures[sig]; !exists {
			return false
		}
	}
	return true
}
