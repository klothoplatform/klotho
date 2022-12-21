package annotation

import (
	"github.com/pelletier/go-toml/v2"
	"go.uber.org/zap"
)

type Directives map[string]interface{}

func ParseDirectives(s string) (Directives, error) {
	var d Directives
	err := toml.Unmarshal([]byte(s), &d)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (d Directives) String(key string) (string, bool) {
	v, ok := d[key]
	if !ok {
		return "", false
	}

	s, ok := v.(string)
	if !ok {
		return "", false
	}

	return s, true
}

// StringArray returns an array of strings (converting a single string to an array of lenth 1).
func (d Directives) StringArray(key string) ([]string, bool) {
	switch v := d[key].(type) {
	case string:
		return []string{v}, true
	case []string:
		return v, true
	case []interface{}:
		list := make([]string, 0, len(v))
		for _, elem := range v {
			if str, ok := elem.(string); ok {
				list = append(list, str)
			} else {
				zap.S().Warnf("skipping non-string (%[1]T) in Directives.StringArray: %[1]v", elem)
			}
		}
		return list, true
	}
	return nil, false
}

func (d Directives) Int(key string) (int, bool) {
	v, ok := d[key]
	if !ok {
		return 0, false
	}
	if v == nil {
		return 0, false
	}
	switch i := v.(type) {
	case int:
		return i, true

	case int32:
		return int(i), true

	case int64:
		return int(i), true

	case uint:
		return int(i), true

	case uint32:
		return int(i), true

	case uint64:
		return int(i), true

	case float64:
		return int(i), true
	}
	return 0, false
}

func (d Directives) Object(key string) Directives {
	v, ok := d[key]
	if !ok {
		return make(Directives)
	}
	m, ok := v.(map[string]interface{})
	if !ok {
		return make(Directives)
	}
	return Directives(m)
}

func (d Directives) Bool(key string) (bool, bool) {
	v, ok := d[key]
	if !ok {
		return false, false
	}
	b, ok := v.(bool)
	if !ok {
		return false, false
	}
	return b, true
}
