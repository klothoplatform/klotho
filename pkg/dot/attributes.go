package dot

import (
	"fmt"
	"sort"
	"strings"
)

func AttributesToString(attribs map[string]string) string {
	if len(attribs) == 0 {
		return ""
	}
	var keys []string
	for k := range attribs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var list []string
	for _, k := range keys {
		v := attribs[k]
		v = strings.ReplaceAll(v, `"`, `\"`)
		list = append(list, fmt.Sprintf(`%s="%s"`, k, v))
	}
	return " [" + strings.Join(list, ", ") + "]"
}
