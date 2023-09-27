package construct2

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

type (
	Properties = map[string]interface{}
)

// SetProperty is a wrapper around [PropertyPath.Set] for convenience.
func (r *Resource) SetProperty(pathStr string, value any) error {
	path, err := r.PropertyPath(pathStr)
	if err != nil {
		return err
	}
	return path.Set(value)
}

// GetProperty is a wrapper around [PropertyPath.Get] for convenience.
func (r *Resource) GetProperty(pathStr string) (any, error) {
	path, err := r.PropertyPath(pathStr)
	if err != nil {
		return nil, err
	}
	return path.Get(), nil
}

// AppendProperty is a wrapper around [PropertyPath.Append] for convenience.
func (r *Resource) AppendProperty(pathStr string, value any) error {
	path, err := r.PropertyPath(pathStr)
	if err != nil {
		return err
	}
	return path.Append(value)
}

// RemoveProperty is a wrapper around [PropertyPath.Remove] for convenience.
func (r *Resource) RemoveProperty(pathStr string, value any) error {
	path, err := r.PropertyPath(pathStr)
	if err != nil {
		return err
	}
	return path.Remove(value)
}

type (
	propertyPathItem interface {
		Get() any
		Set(value any) error
		Remove(value any) error
		Append(value any) error

		parent() propertyPathItem
	}

	// PropertyPath represents a path into a resource's properties. See [Resource.PropertyPath] for
	// more information.
	PropertyPath []propertyPathItem

	mapKeyPathItem struct {
		_parent propertyPathItem
		m       reflect.Value
		key     reflect.Value
	}

	arrayIndexPathItem struct {
		_parent propertyPathItem
		a       reflect.Value
		index   int
	}
)

func splitPath(path string) []string {
	var parts []string
	var delim string
	for path != "" {
		partIdx := strings.IndexAny(path, ".[")
		var part string
		if partIdx == -1 {
			part = delim + path
			path = ""
		} else {
			part = delim + path[:partIdx]
			delim = path[partIdx : partIdx+1]
			path = path[partIdx+1:]
		}
		parts = append(parts, part)
	}
	return parts
}

// PropertyPath interprets a string path to index (potentially deeply) into [Resource.Properties]
// which can be used to get, set, append, or remove values.
func (r *Resource) PropertyPath(pathStr string) (PropertyPath, error) {
	pathParts := splitPath(pathStr)
	if len(pathParts) == 0 {
		return nil, fmt.Errorf("empty path")
	}
	path := make(PropertyPath, len(pathParts))
	value := reflect.ValueOf(r.Properties)
	for i, part := range pathParts {
		switch part[0] {
		case '.':
			part = part[1:]
			fallthrough
		default:
			for value.Kind() == reflect.Interface || value.Kind() == reflect.Ptr {
				value = value.Elem()
			}
			if value.Kind() != reflect.Map {
				return nil, &PropertyPathError{
					Path:  pathParts[:i-1],
					Cause: fmt.Errorf("expected map, got %s", value.Type()),
				}
			}
			item := mapKeyPathItem{
				m:   value,
				key: reflect.ValueOf(part),
			}
			if i > 0 {
				item._parent = path[i-1]
			}
			path[i] = item
			value = value.MapIndex(item.key)

		case '[':
			for value.Kind() == reflect.Interface || value.Kind() == reflect.Ptr {
				value = value.Elem()
			}
			if value.Kind() != reflect.Slice && value.Kind() != reflect.Array {
				return nil, &PropertyPathError{
					Path:  pathParts[:i-1],
					Cause: fmt.Errorf("expected array, got %s", value.Type()),
				}
			}
			if len(part) < 2 || part[len(part)-1] != ']' {
				return nil, &PropertyPathError{
					Path:  pathParts[:i],
					Cause: fmt.Errorf("invalid array index format, got %q", part),
				}
			}
			idxStr := part[1 : len(part)-1]
			idx, err := strconv.Atoi(idxStr)
			if err != nil {
				return nil, &PropertyPathError{Path: pathParts[:i], Cause: err}
			}
			if idx < 0 || idx >= value.Len() {
				return nil, &PropertyPathError{
					Path:  pathParts[:i],
					Cause: fmt.Errorf("array index out of bounds: %d (length %d)", idx, value.Len()),
				}
			}
			path[i] = arrayIndexPathItem{
				_parent: path[i-1],
				a:       value,
				index:   idx,
			}
			value = value.Index(idx)
		}
	}
	return path, nil
}

type PropertyPathError struct {
	Path  []string
	Cause error
}

func (e *PropertyPathError) Error() string {
	return fmt.Sprintf("error in path %s: %v",
		strings.Join(e.Path, ""),
		e.Cause,
	)
}

func itemToPath(i propertyPathItem) []string {
	path, ok := i.(PropertyPath)
	if ok {
		return path.Parts()
	}
	var items []propertyPathItem
	for i != nil {
		items = append(items, i)
		i = i.parent()
	}
	// reverse items so that we get the path in the correct order
	for idx := 0; idx < len(items)/2; idx++ {
		items[idx], items[len(items)-idx-1] = items[len(items)-idx-1], items[idx]
	}
	return PropertyPath(items).Parts()
}

func (e *PropertyPathError) Unwrap() error {
	return e.Cause
}

func pathPanicRecover(i propertyPathItem, err *error) {
	if r := recover(); r != nil {
		rerr, ok := r.(error)
		if !ok {
			rerr = fmt.Errorf("panic: %v", r)
		}
		*err = &PropertyPathError{
			Path:  itemToPath(i),
			Cause: rerr,
		}
	}
}

func (i mapKeyPathItem) Set(value any) (err error) {
	defer pathPanicRecover(i, &err)
	i.m.SetMapIndex(i.key, reflect.ValueOf(value))
	return nil
}

func (i mapKeyPathItem) Append(value any) (err error) {
	defer pathPanicRecover(i, &err)

	a := i.m.MapIndex(i.key)
	for a.Kind() == reflect.Interface || a.Kind() == reflect.Ptr {
		a = a.Elem()
	}
	if a.Kind() != reflect.Slice && a.Kind() != reflect.Array {
		return &PropertyPathError{
			Path:  itemToPath(i),
			Cause: fmt.Errorf("expected array for append, got %s", a.Type()),
		}
	}
	i.m.SetMapIndex(i.key, reflect.Append(a, reflect.ValueOf(value)))
	return nil
}

func arrRemoveByValue(arr reflect.Value, value reflect.Value) (reflect.Value, error) {
	newArr := reflect.MakeSlice(arr.Type(), 0, arr.Len())
	for i := 0; i < arr.Len(); i++ {
		item := arr.Index(i)
		if !item.Equal(value) {
			newArr = reflect.Append(newArr, item)
		}
	}
	if newArr.Len() == arr.Len() {
		return arr, fmt.Errorf("value %v not found in array", value)
	}
	return newArr, nil
}

func (i mapKeyPathItem) Remove(value any) (err error) {
	defer pathPanicRecover(i, &err)
	if value == nil {
		i.m.SetMapIndex(i.key, reflect.Value{})
		return nil
	}
	arr := i.m.MapIndex(i.key)
	for arr.Kind() == reflect.Interface || arr.Kind() == reflect.Ptr {
		arr = arr.Elem()
	}
	if arr.Kind() != reflect.Slice && arr.Kind() != reflect.Array {
		return &PropertyPathError{
			Path:  itemToPath(i),
			Cause: fmt.Errorf("for non-nil value'd (%v), must be array (got %s) to remove by value", value, arr.Type()),
		}
	}
	newArr, err := arrRemoveByValue(arr, reflect.ValueOf(value))
	if err != nil {
		return &PropertyPathError{Path: itemToPath(i), Cause: err}
	}
	i.m.SetMapIndex(i.key, newArr)
	return nil
}

func (i mapKeyPathItem) Get() any {
	v := i.m.MapIndex(i.key)
	if !v.IsValid() {
		return nil
	}
	return v.Interface()
}

func (i mapKeyPathItem) parent() propertyPathItem {
	return i._parent
}

func (i arrayIndexPathItem) Set(value any) (err error) {
	defer pathPanicRecover(i, &err)
	i.a.Index(i.index).Set(reflect.ValueOf(value))
	return nil
}

func (i arrayIndexPathItem) Append(value any) (err error) {
	defer pathPanicRecover(i, &err)
	a := i.a.Index(i.index)
	for a.Kind() == reflect.Interface || a.Kind() == reflect.Ptr {
		a = a.Elem()
	}
	if a.Kind() != reflect.Slice && a.Kind() != reflect.Array {
		return &PropertyPathError{
			Path:  itemToPath(i),
			Cause: fmt.Errorf("expected array for append, got %s", a),
		}
	}
	i.a.Index(i.index).Set(reflect.Append(a, reflect.ValueOf(value)))
	return nil
}

func (i arrayIndexPathItem) Remove(value any) (err error) {
	defer pathPanicRecover(i, &err)
	if value == nil {
		i.a = reflect.AppendSlice(i.a.Slice(0, i.index), i.a.Slice(i.index+1, i.a.Len()))
		return i._parent.Set(i.a.Interface())
	}

	arr := i.a.Index(i.index)
	for arr.Kind() == reflect.Interface || arr.Kind() == reflect.Ptr {
		arr = arr.Elem()
	}
	if arr.Kind() != reflect.Slice && arr.Kind() != reflect.Array {
		return &PropertyPathError{
			Path:  itemToPath(i),
			Cause: fmt.Errorf("for non-nil value'd (%v), must be array (got %s) to remove by value", value, arr.Type()),
		}
	}
	newArr, err := arrRemoveByValue(arr, reflect.ValueOf(value))
	if err != nil {
		return &PropertyPathError{Path: itemToPath(i), Cause: err}
	}
	arr.Set(newArr)
	return nil
}

func (i arrayIndexPathItem) Get() any {
	return i.a.Index(i.index).Interface()
}

func (i arrayIndexPathItem) parent() propertyPathItem {
	return i._parent
}

// Set sets the value at this path item.
func (i PropertyPath) Set(value any) error {
	return i[len(i)-1].Set(value)
}

// Append appends a value to the item. Only supported on array items.
func (i PropertyPath) Append(value any) error {
	return i[len(i)-1].Append(value)
}

// Remove removes the value at this path item. If value is nil, it is interpreted
// to remove the item itself. Non-nil value'd remove is only supported on array items, to
// remove a value from the array.
func (i PropertyPath) Remove(value any) error {
	return i[len(i)-1].Remove(value)
}

// Get returns the value at this path item.
func (i PropertyPath) Get() any {
	return i[len(i)-1].Get()
}

func (i PropertyPath) parent() propertyPathItem {
	return i[len(i)-1].parent()
}

func (i PropertyPath) Parts() []string {
	parts := make([]string, len(i))
	for idx, item := range i {
		switch item := item.(type) {
		case mapKeyPathItem:
			key := item.key.String()
			if idx > 0 {
				key = "." + key
			}
			parts[idx] = key
		case arrayIndexPathItem:
			parts[idx] = fmt.Sprintf("[%d]", item.index)
		}
	}
	return parts
}

func (i PropertyPath) String() string {
	return strings.Join(i.Parts(), "")
}

type WalkPropertiesFunc func(path PropertyPath, err error) error

func mapKeys(m reflect.Value) ([]reflect.Value, error) {
	if m.Type().Key().Kind() != reflect.String {
		return nil, fmt.Errorf("expected map[string]..., got %s", m.Type())
	}

	keys := m.MapKeys()
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].String() < keys[j].String()
	})
	return keys, nil
}

var SkipProperty = fmt.Errorf("skip property")

func (r *Resource) WalkProperties(fn WalkPropertiesFunc) error {
	queue := make([]PropertyPath, len(r.Properties))
	props := reflect.ValueOf(r.Properties)
	keys, _ := mapKeys(props)
	for i, k := range keys {
		queue[i] = PropertyPath{mapKeyPathItem{m: props, key: k}}
	}

	var err error
	var item PropertyPath
	for len(queue) > 0 {
		item, queue = queue[0], queue[1:]

		err = fn(item, err)
		if err == StopWalk {
			return nil
		}
		if err == SkipProperty {
			continue
		}

		v := reflect.ValueOf(item.Get())
		switch v.Kind() {
		case reflect.Map:
			keys, err := mapKeys(v)
			if err != nil {
				return err
			}
			for _, k := range keys {
				queue = append(queue, append(item, mapKeyPathItem{m: v, key: k}))
			}

		case reflect.Array, reflect.Slice:
			for i := 0; i < v.Len(); i++ {
				queue = append(queue, append(item, arrayIndexPathItem{a: v, index: i}))
			}
		}
	}
	return err
}
