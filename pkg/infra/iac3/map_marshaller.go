package iac3

import (
	"fmt"
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

	TsMap map[string]any

	TsList []any
)

func (m TsMap) String() string {
	buf := strings.Builder{}
	buf.WriteRune('{')
	keys := len(m)
	i := 0
	for k, v := range m {
		buf.WriteString(fmt.Sprintf("%s: %v", k, v))
		if i <= keys {
			buf.WriteRune(',')
		}
		i++
	}
	buf.WriteRune('}')
	return buf.String()
}

func (l TsList) String() string {
	buf := strings.Builder{}
	buf.WriteRune('[')
	for i, v := range l {
		fmt.Fprintf(&buf, "%v", v)
		if i < len(l) {
			buf.WriteRune(',')
		}
	}
	buf.WriteRune(']')
	return buf.String()
}
