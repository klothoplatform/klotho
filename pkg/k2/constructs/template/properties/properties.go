package properties

import (
	"bytes"
	"encoding"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"text/template"

	"github.com/klothoplatform/klotho/pkg/k2/constructs/template/property"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/klothoplatform/klotho/pkg/templateutils"
)

type (
	DefaultExecutionContext struct{}
)

// UnmarshalFunc decodes data into the supplied pointer, v
type UnmarshalFunc func(data *bytes.Buffer, v any) error

func (d DefaultExecutionContext) ExecuteUnmarshal(tmpl string, data any, v any) error {
	parsedTemplate, err := template.New("tmpl").Funcs(templateutils.WithCommonFuncs(template.FuncMap{})).Parse(tmpl)
	if err != nil {
		return err
	}

	return ExecuteTemplateUnmarshal(parsedTemplate, data, v, d.Unmarshal)
}

func (d DefaultExecutionContext) Unmarshal(data *bytes.Buffer, v any) error {
	return UnmarshalAny(data, v)
}

// ExecuteTemplateUnmarshal executes the [template.Template], t, using data and unmarshals the value into v
func ExecuteTemplateUnmarshal(
	t *template.Template,
	data any,
	v any,
	unmarshal UnmarshalFunc,
) error {
	buf := new(bytes.Buffer)
	if err := t.Execute(buf, data); err != nil {
		return err
	}

	if err := unmarshal(buf, v); err != nil {
		return fmt.Errorf("cannot decode template result '%s' into %T", buf, v)
	}

	return nil
}

func UnmarshalJSON(data *bytes.Buffer, outputRefValue any) error {
	dec := json.NewDecoder(data)
	return dec.Decode(outputRefValue)
}

// UnmarshalAny decodes the template result into a primitive or a struct that implements [encoding.TextUnmarshaler].
// As a fallback, it tries to unmarshal the result using [json.Unmarshal].
// If v is a pointer, it will be set to the decoded value.
func UnmarshalAny(data *bytes.Buffer, v any) error {
	// trim the spaces, so you don't have to sprinkle the templates with `{{-` and `-}}` (the `-` trims spaces)
	bstr := strings.TrimSpace(data.String())
	switch value := v.(type) {
	case *string:
		*value = bstr
		return nil

	case *[]byte:
		*value = []byte(bstr)
		return nil

	case *bool:
		result := strings.ToLower(bstr)
		// If the input (eg 'field') is nil and the 'if' statement just uses '{{ inputs "field" }}',
		// then the string result will be '<no value>'.
		// Make sure we don't interpret that as a true condition.
		*value = result != "" && result != "<no value>" && strings.ToLower(result) != "false"
		return nil
	case *int:
		i, err := strconv.Atoi(bstr)
		if err != nil {
			return err
		}
		*value = i
		return nil
	case *float64:
		f, err := strconv.ParseFloat(bstr, 64)
		if err != nil {
			return err
		}
		*value = f
		return nil
	case *float32:
		f, err := strconv.ParseFloat(bstr, 32)
		if err != nil {
			return err
		}
		*value = float32(f)
		return nil

	case encoding.TextUnmarshaler:
		// notably, this handles `construct.ResourceId` and `construct.IaCValue`
		return value.UnmarshalText([]byte(bstr))
	}

	resultStr := reflect.ValueOf(data.String())
	valueRefl := reflect.ValueOf(v).Elem()
	if resultStr.Type().AssignableTo(valueRefl.Type()) {
		// this covers alias types like `type MyString string`
		valueRefl.Set(resultStr)
		return nil
	}

	err := json.Unmarshal([]byte(bstr), v)
	if err == nil {
		return nil
	}

	return err

}

func ExecuteUnmarshalAsURN(ctx property.ExecutionContext, tmpl string, data any) (model.URN, error) {
	var selector model.URN
	err := ctx.ExecuteUnmarshal(tmpl, data, &selector)
	if err != nil {
		return selector, err
	}
	if selector.IsZero() {
		return selector, fmt.Errorf("selector '%s' is zero", tmpl)
	}
	return selector, nil
}
