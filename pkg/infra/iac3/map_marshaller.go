package iac3

import (
	"fmt"
	"sort"
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
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	i := 0
	for _, k := range keys {
		v := m[k]
		buf.WriteString(fmt.Sprintf("%s: %v", k, v))
		if i < len(m)-1 {
			buf.WriteString(", ")
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
		if i < len(l)-1 {
			buf.WriteString(", ")
		}
	}
	buf.WriteRune(']')
	return buf.String()
}
