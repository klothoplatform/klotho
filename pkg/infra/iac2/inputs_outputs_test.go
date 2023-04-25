package iac2

import (
	"io/fs"
	"reflect"
	"strings"
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	assert "github.com/stretchr/testify/assert"
)

// TestKnownTemplates performs several tests to make sure that our Go structs match up with the factory.ts templates.
//
// For each known type, it checks:
//   - That there's a template for that struct
//   - That the template has a valid "output" type
//   - That for each input defined in the template's Args: (a) there is a corresponding field in the Go struct, and (b)
//     that the Go struct's field type matches with the Arg field type.
//
// To do the field-matching, we look at the Go struct's field type, and compute from it the expected TypeScript type.
// For primitives (int, bool, string, etc) we just convert it to the corresponding TypeScript primitive. For structs,
// we look up that struct's template and expect whatever the output of that template is. See the [iac2] package docs
// for more ("Why a template for a template?").
//
// We don't check the template's return expression, because we assume that if it has a valid output type, it'll also
// have a valid return expression. (Otherwise, our separate tsc checks will fail.)
//
// With all that done, we also check that we've validated all the structs in pkg/provider/aws/. To do this,
// we use the reflective [packages.Load] to find all the types within that package, and then filter down to those types
// that conform to core.Resource. Then we simply check that each one of those is in the list of types we checked.
func TestKnownTemplates(t *testing.T) {
	allResources := resources.ListAll()
	allResources = append(allResources,
		&KubernetesProvider{},
		&RolePolicyAttachment{},
		&RouteTableAssociation{},
		&SecurityGroupRule{},
	)

	tp := standardTemplatesProvider()
	testedTypes := make(coretesting.TypeRefSet)
	for _, res := range allResources {
		testedTypes.Add(res)
		baseResourceType := reflect.TypeOf(res)
		resType := baseResourceType
		for resType.Kind() == reflect.Pointer {
			resType = resType.Elem()
		}
		t.Run(resType.String(), func(t *testing.T) {
			var tmpl ResourceCreationTemplate

			tmplFound := t.Run("template exists", func(t *testing.T) {
				assert := assert.New(t)
				found, err := tp.getTemplate(res)
				if !assert.NoError(err) {
					return
				}
				tmpl = found
			})
			if !tmplFound {
				return
			}
			t.Run("output", func(t *testing.T) {
				assert := assert.New(t)
				assert.NotEmpty(tmpl.OutputType)
			})

			t.Run("inputs", func(t *testing.T) {
				for inputName, inputTsType := range tmpl.InputTypes {
					if inputName == "dependsOn" || inputName == "protect" || inputName == "awsProfile" {
						continue
					}
					t.Run(inputName, func(t *testing.T) {
						assert := assert.New(t)

						var inputType reflect.Type

						if field, fieldFound := resType.FieldByName(inputName); fieldFound {
							assert.Truef(field.IsExported(), `field is not exported`, field.Name)
							inputType = field.Type
						} else {
							method, methodFound := resType.MethodByName(inputName)
							if !methodFound {
								// Fallback to the base resource type in case the method is defined on a pointer receiver
								method, methodFound = baseResourceType.MethodByName(inputName)
							}
							if methodFound {
								assert.Truef(method.IsExported(), `method '%s' is not exported`, method.Name)
								assert.Truef(method.Type.NumIn() == 1, `method '%s' has more than one (%d) input`, method.Name, method.Type.NumIn())
								assert.Truef(method.Type.NumOut() > 0, `method '%s' has no output`, method.Name)
								assert.Truef(method.Type.NumOut() <= 2, `method '%s' has too many (%d) output`, method.Name, method.Type.NumOut())
								inputType = method.Type.Out(0)
							}
						}

						if !assert.NotNil(inputType, `%T missing field/method '%s'`, res, inputName) {
							return
						}

						if inputType.Kind() == reflect.Interface && inputType == reflect.TypeOf((*core.Resource)(nil)).Elem() {
							return
						}
						// avoids fields which use nested template or document functionality
						if inputType.Kind() == reflect.Struct || inputType.Kind() == reflect.Pointer && inputType != reflect.TypeOf((*core.Resource)(nil)).Elem() || inputType != reflect.TypeOf((*core.IaCValue)(nil)).Elem() {
							return
						}

						expectedType := &strings.Builder{}
						if err := buildExpectedTsType(expectedType, tp, inputType); !assert.NoError(err) {
							return
						}
						assert.NotEmpty(expectedType, `couldn't determine expected type'`)
						assert.Equal(expectedType.String(), inputTsType, `field type`)

					})
				}
			})
		})
	}
	t.Run("all types tested", func(t *testing.T) {
		for _, ref := range coretesting.FindAllResources(assert.New(t), allResources) {
			t.Run(ref.Name, func(t *testing.T) {
				testedTypes.Check(t, ref, `struct implements core.Resource but isn't tested; add it to this test's '"allResources" var`)
			})
		}
	})
	t.Run("all templates used", func(t *testing.T) {
		usedTemplates := make(map[string]struct{})
		for _, res := range allResources {
			usedTemplates[camelToSnake(structName(res))] = struct{}{}
		}
		err := fs.WalkDir(tp.templates, ".", func(path string, d fs.DirEntry, err error) error {
			if path == "." {
				return nil
			}
			t.Run(path, func(t *testing.T) {
				assert := assert.New(t)
				assert.Contains(usedTemplates, path, `template isn't used; add a core.Resource implementation for it`)
			})
			return fs.SkipDir
		})
		if !assert.New(t).NoError(err) {
			return
		}
	})
}

// buildExpectedTsType converts a Go type to an expected TypeScript type. For example, a map[string]int would translate
// to Record<string, number>.
func buildExpectedTsType(out *strings.Builder, tp *templatesProvider, t reflect.Type) error {

	// ok, general cases now
	switch t.Kind() {
	case reflect.Bool:
		out.WriteString(`boolean`)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		out.WriteString(`number`)
	case reflect.String:
		out.WriteString(`string`)
	case reflect.Struct:
		if t == reflect.TypeOf((*core.IaCValue)(nil)).Elem() {
			out.WriteString("pulumi.Output<string>")
		} else {
			res, err := tp.getTemplateForType(t.Name())
			if err != nil {
				return err
			}
			out.WriteString(res.OutputType)
		}
	case reflect.Array, reflect.Slice:
		err := buildExpectedTsType(out, tp, t.Elem())
		if err != nil {
			return err
		}
		out.WriteString("[]")
	case reflect.Map:
		out.WriteString("Record<")
		err := buildExpectedTsType(out, tp, t.Key())
		if err != nil {
			return err
		}
		out.WriteString(", ")
		err = buildExpectedTsType(out, tp, t.Elem())
		if err != nil {
			return err
		}
		out.WriteRune('>')
	case reflect.Pointer:
		// Pointer happens when the value is inside a map, slice, or array. Basically, the reflected type is
		// interface{},instead of being the actual type. So, we basically pull the item out of the collection, and then
		// reflect on it directly.
		err := buildExpectedTsType(out, tp, t.Elem())
		if err != nil {
			return err
		}
	case reflect.Interface:
		// Similar to Pointer above; but specifically when the map/slice's type is "any". For example,
		// `map[string]int` will hit the `reflect.Pointer`case for the value type, but `map[string]any` will his here.
		out.WriteString(`pulumi.Output<any>`)
	}
	return nil
}
