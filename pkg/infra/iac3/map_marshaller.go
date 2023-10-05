package iac3

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/iancoleman/strcase"
)

func (tc *TemplatesCompiler) listMarshaller(arg any, templateArg *Arg) (string, error) {
	val := reflect.ValueOf(arg)

	buf := strings.Builder{}
	buf.WriteRune('[')
	for i := 0; i < val.Len(); i++ {
		output, err := tc.convertArg(val.Index(i).Interface(), templateArg)
		if err != nil {
			return "", err
		}
		buf.WriteString(fmt.Sprintf("%v", output))
		if i < (val.Len() - 1) {
			buf.WriteRune(',')
		}
	}
	buf.WriteRune(']')
	return buf.String(), nil
}

func (tc *TemplatesCompiler) mapMarshaller(arg any, templateArg *Arg) (string, error) {
	val := reflect.ValueOf(arg)
	buf := strings.Builder{}
	buf.WriteRune('{')
	for i, key := range val.MapKeys() {
		if !val.MapIndex(key).IsValid() || val.MapIndex(key).IsNil() {
			continue
		}
		keyStr, found := key.Interface().(string)
		if !found {
			return "", fmt.Errorf("map key is not a string")
		}
		keyResult := strcase.ToLowerCamel(keyStr)
		if templateArg != nil && templateArg.Wrapper == string(CamelCaseWrapper) {
			keyResult = strcase.ToCamel(keyStr)
		} else if templateArg != nil && templateArg.Wrapper == string(ModelCaseWrapper) {
			keyResult = keyStr
		}
		buf.WriteString(keyResult)

		buf.WriteRune(':')
		output, err := tc.convertArg(val.MapIndex(key).Interface(), templateArg)
		if err != nil {
			return "", err
		}
		buf.WriteString(fmt.Sprintf("%v", output))
		if i < (len(val.MapKeys()) - 1) {
			buf.WriteRune(',')
		}
	}
	buf.WriteRune('}')
	return buf.String(), nil
}
