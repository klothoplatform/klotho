package property

import (
	"errors"
	"fmt"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/set"
	"reflect"
	"sort"
	"strings"
)

// PropertyMap is a map of properties that can be used to represent complex data structures in a template
// Wrap this in a struct that implements the [Properties] interface when using it in a template
type PropertyMap map[string]Property

func (p PropertyMap) Clone() PropertyMap {
	newProps := make(PropertyMap, len(p))
	for k, v := range p {
		newProps[k] = v.Clone()
	}
	return newProps
}

func (p PropertyMap) Get(key string) (Property, bool) {
	value, exists := p[key]
	return value, exists
}

func (p PropertyMap) Set(key string, value Property) {
	p[key] = value
}

func (p PropertyMap) Remove(key string) {
	delete(p, key)
}

func (p PropertyMap) ForEach(c construct.Properties, f func(p Property) error) error {
	queue := []PropertyMap{p}
	var props PropertyMap
	var errs error
	for len(queue) > 0 {
		props, queue = queue[0], queue[1:]

		propKeys := make([]string, 0, len(props))
		for k := range props {
			propKeys = append(propKeys, k)
		}
		sort.Strings(propKeys)

		for _, key := range propKeys {
			prop := props[key]
			err := f(prop)
			if err != nil {
				if errors.Is(err, ErrStopWalk) {
					return nil
				}
				errs = errors.Join(errs, err)
				continue
			}

			if strings.HasPrefix(prop.Type(), "list") || strings.HasPrefix(prop.Type(), "set") {
				p, err := c.GetProperty(prop.Details().Path)
				if err != nil || p == nil {
					continue
				}
				// Because lists/sets will start as empty, do not recurse into their sub-properties if it's not set.
				// To allow for defaults within list objects and operational rules to be run,
				// we will look inside the property to see if there are values.
				if strings.HasPrefix(prop.Type(), "list") {
					length := reflect.ValueOf(p).Len()
					for i := 0; i < length; i++ {
						subProperties := make(PropertyMap)
						for subK, subProp := range prop.SubProperties() {
							propTemplate := subProp.Clone()
							ReplacePath(propTemplate, prop.Details().Path, fmt.Sprintf("%s[%d]", prop.Details().Path, i))
							subProperties[subK] = propTemplate
						}
						if len(subProperties) > 0 {
							queue = append(queue, subProperties)
						}
					}
				} else if strings.HasPrefix(prop.Type(), "set") {
					hs, ok := p.(set.HashedSet[string, any])
					if !ok {
						errs = errors.Join(errs, fmt.Errorf("could not cast property to set"))
						continue
					}
					for i := range hs.ToSlice() {
						subProperties := make(PropertyMap)
						for subK, subProp := range prop.SubProperties() {
							propTemplate := subProp.Clone()
							ReplacePath(propTemplate, prop.Details().Path, fmt.Sprintf("%s[%d]", prop.Details().Path, i))
							subProperties[subK] = propTemplate
						}
						if len(subProperties) > 0 {
							queue = append(queue, subProperties)
						}
					}

				}
			} else if prop.SubProperties() != nil {
				queue = append(queue, prop.SubProperties())
			}
		}
	}
	return errs
}

func GetProperty(properties PropertyMap, path string) Property {
	fields := strings.Split(path, ".")
FIELDS:
	for i, field := range fields {
		currFieldName := strings.Split(field, "[")[0]
		found := false
		for name, property := range properties {
			if name != currFieldName {
				continue
			}
			found = true
			if len(fields) == i+1 {
				// use a clone resource so we can modify the name in case anywhere in the path
				// has index strings or map keys
				clone := property.Clone()
				details := clone.Details()
				details.Path = path
				return clone
			} else {
				properties = property.SubProperties()
				if len(properties) == 0 {
					if mp, ok := property.(MapProperty); ok {
						clone := mp.Value().Clone()
						details := clone.Details()
						details.Path = path
						return clone
					} else if cp, ok := property.(CollectionProperty); ok {
						clone := cp.Item().Clone()
						details := clone.Details()
						details.Path = path
						return clone
					}
				}
			}
			continue FIELDS
		}
		if !found {
			return nil
		}
	}
	return nil
}
