package iac3

import (
	"fmt"
	"reflect"
	"strings"
)

type (
	MapMarshaller interface {
		Map() map[string]any
		String() string
		SetKey(val any)
	}

	ListMarshaller interface {
		List() []any
		String() string
		Append(val any)
	}

	TsMap struct {
		m map[string]any
	}

	TsList struct {
		l []any
	}
)

func (m *TsMap) Map() map[string]any {
	return m.m
}

func (m *TsMap) SetKey(key string, val any) {
	m.m[key] = val
}

func (m *TsMap) String() string {
	val := reflect.ValueOf(m.m)
	buf := strings.Builder{}
	buf.WriteRune('{')
	for i, key := range val.MapKeys() {
		if !val.MapIndex(key).IsValid() || val.MapIndex(key).IsNil() {
			continue
		}
		keyStr, found := key.Interface().(string)
		if !found {
			panic("map key is not a string")
		}
		buf.WriteString(keyStr)

		buf.WriteRune(':')
		buf.WriteString(fmt.Sprintf("%v", val.MapIndex(key).Interface()))
		if i < (len(val.MapKeys()) - 1) {
			buf.WriteRune(',')
		}
	}
	buf.WriteRune('}')
	return buf.String()
}

func (l *TsList) List() []any {
	return l.l
}

func (l *TsList) Append(val any) {
	l.l = append(l.l, val)
}

func (l *TsList) String() string {

	val := reflect.ValueOf(l.l)

	buf := strings.Builder{}
	buf.WriteRune('[')
	for i := 0; i < val.Len(); i++ {
		buf.WriteString(fmt.Sprintf("%v", val.Index(i).Interface()))
		if i < (val.Len() - 1) {
			buf.WriteRune(',')
		}
	}
	buf.WriteRune(']')
	return buf.String()
}
