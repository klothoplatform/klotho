package construct2

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/klothoplatform/klotho/pkg/yaml_util"
)

type (
	Properties map[string]any
)

// SetProperty is a wrapper around [PropertyPath.Set] for convenience.
func (r *Resource) SetProperty(pathStr string, value any) error {
	if r.Properties == nil {
		r.Properties = Properties{}
	}
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
	PropertyPathItem interface {
		Get() any
		Set(value any) error
		Remove(value any) error
		Append(value any) error

		parent() PropertyPathItem
	}

	PropertyKVItem interface {
		Key() PropertyPathItem
	}

	// PropertyPath represents a path into a resource's properties. See [Resource.PropertyPath] for
	// more information.
	PropertyPath []PropertyPathItem

	mapValuePathItem struct {
		_parent PropertyPathItem
		m       reflect.Value
		key     reflect.Value
	}

	mapKeyPathItem mapValuePathItem

	arrayIndexPathItem struct {
		_parent PropertyPathItem
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
			if value.IsValid() && value.Kind() != reflect.Map {
				return nil, &PropertyPathError{
					Path:  pathParts[:i-1],
					Cause: fmt.Errorf("expected map, got %s", value.Type()),
				}
			}
			item := mapValuePathItem{
				m:   value,
				key: reflect.ValueOf(part),
			}
			if i > 0 {
				item._parent = path[i-1]
			}
			path[i] = item
			if value.IsValid() {
				value = value.MapIndex(item.key)
			}
		case '[':
			for value.Kind() == reflect.Interface || value.Kind() == reflect.Ptr {
				value = value.Elem()
			}
			if value.IsValid() && value.Kind() != reflect.Slice && value.Kind() != reflect.Array {
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
			if value.IsValid() {
				value = value.Index(idx)
			}
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

func itemToPath(i PropertyPathItem) []string {
	path, ok := i.(PropertyPath)
	if ok {
		return path.Parts()
	}
	var items []PropertyPathItem
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

func pathPanicRecover(i PropertyPathItem, operation string, err *error) {
	if r := recover(); r != nil {
		rerr, ok := r.(error)
		if !ok {
			rerr = fmt.Errorf("panic: %v", r)
		}
		*err = &PropertyPathError{
			Path:  itemToPath(i),
			Cause: fmt.Errorf("recovered panic during '%s': %w", operation, rerr),
		}
	}
}

func (i mapValuePathItem) Set(value any) (err error) {
	defer pathPanicRecover(i, "Set on map", &err)
	if !i.m.IsValid() {
		i.m = reflect.MakeMap(reflect.MapOf(i.key.Type(), reflect.TypeOf((*any)(nil)).Elem()))
		err = i._parent.Set(i.m.Interface())
		if err != nil {
			return
		}
	}
	i.m.SetMapIndex(i.key, reflect.ValueOf(value))
	return nil
}

func appendValue(appendTo reflect.Value, value reflect.Value) (reflect.Value, error) {
	a := appendTo
	for a.Kind() == reflect.Interface || a.Kind() == reflect.Ptr {
		a = a.Elem()
	}
	if !a.IsValid() {
		// Appending to empty, create a new slice or map based on what value's type.
		switch value.Kind() {
		case reflect.Slice, reflect.Array:
			// append(nil, []T{...}} => []T{...}
			a = reflect.MakeSlice(value.Type(), 0, value.Len())

		case reflect.Map:
			// append(nil, map[K]V{...}) => map[K]V{...}
			a = reflect.MakeMap(reflect.MapOf(value.Type().Key(), value.Type().Elem()))

		default:
			// append(nil, T) => []T{...}
			a = reflect.MakeSlice(reflect.SliceOf(value.Type()), 0, 1)
		}
	}

	switch a.Kind() {
	case reflect.Slice, reflect.Array:
		var values []reflect.Value
		if (value.Kind() == reflect.Slice || value.Kind() == reflect.Array) &&
			value.Type().Elem().AssignableTo(a.Type().Elem()) {
			// append(a []T, b []T) => []T{a..., b...}

			values = make([]reflect.Value, value.Len())
			for i := 0; i < value.Len(); i++ {
				values[i] = value.Index(i)
			}
		} else if value.Type().AssignableTo(a.Type().Elem()) {
			// append(a []T, b T) => []T{a..., b}
			values = []reflect.Value{value}
		} else {
			return a, fmt.Errorf("expected %s or []%[1]s value for append, got %s", a.Type().Elem(), value.Type())
		}
		return reflect.Append(a, values...), nil

	case reflect.Map:
		aType := a.Type()
		valType := value.Type()
		if valType.Kind() != reflect.Map {
			return a, fmt.Errorf("expected map value for append, got %s", valType)
		}
		if !valType.Key().AssignableTo(aType.Key()) {
			return a, fmt.Errorf("expected map key type %s, got %s", aType.Key(), valType.Key())
		}
		if !valType.Elem().AssignableTo(aType.Elem()) {
			return a, fmt.Errorf("expected map value type %s, got %s", aType.Elem(), valType.Elem())
		}
		for _, key := range value.MapKeys() {
			a.SetMapIndex(key, value.MapIndex(key))
		}
		return a, nil
	}
	return a, fmt.Errorf("expected array or map destination for append, got %s", a.Kind())
}

func (i mapValuePathItem) Append(value any) (err error) {
	defer pathPanicRecover(i, "Append on map", &err)

	kv := i.m.MapIndex(i.key)
	appended, err := appendValue(kv, reflect.ValueOf(value))
	if err != nil {
		return &PropertyPathError{Path: itemToPath(i), Cause: err}
	}
	i.m.SetMapIndex(i.key, appended)
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

func (i mapValuePathItem) Remove(value any) (err error) {
	defer pathPanicRecover(i, "Remove on map", &err)
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

func (i mapValuePathItem) Get() any {
	if !i.m.IsValid() {
		return nil
	}
	v := i.m.MapIndex(i.key)
	if !v.IsValid() {
		return nil
	}
	return v.Interface()
}

func (i mapValuePathItem) parent() PropertyPathItem {
	return i._parent
}

func (i mapValuePathItem) Key() PropertyPathItem {
	return mapKeyPathItem(i)
}

func (i mapKeyPathItem) Get() any {
	return i.key.Interface()
}

func (i mapKeyPathItem) Set(value any) (err error) {
	defer pathPanicRecover(i, "Set on map key", &err)
	mapValue := i.m.MapIndex(i.key)
	i.m.SetMapIndex(i.key, reflect.Value{})
	i.m.SetMapIndex(reflect.ValueOf(value), mapValue)
	return nil
}

func (i mapKeyPathItem) Append(value any) (err error) {
	return &PropertyPathError{
		Path:  itemToPath(i),
		Cause: fmt.Errorf("cannot append to map key"),
	}
}

func (i mapKeyPathItem) Remove(value any) (err error) {
	defer pathPanicRecover(i, "Remove on map key", &err)
	i.m.SetMapIndex(i.key, reflect.Value{})
	return nil
}

func (i mapKeyPathItem) parent() PropertyPathItem {
	return i._parent
}

func (i arrayIndexPathItem) Set(value any) (err error) {
	defer pathPanicRecover(i, "Set on array", &err)
	i.a.Index(i.index).Set(reflect.ValueOf(value))
	return nil
}

func (i arrayIndexPathItem) Append(value any) (err error) {
	defer pathPanicRecover(i, "Append on array", &err)
	ival := i.a.Index(i.index)
	appended, err := appendValue(ival, reflect.ValueOf(value))
	if err != nil {
		return &PropertyPathError{Path: itemToPath(i), Cause: err}
	}
	ival.Set(appended)
	return nil
}

func (i arrayIndexPathItem) Remove(value any) (err error) {
	defer pathPanicRecover(i, "Remove on array", &err)
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

func (i arrayIndexPathItem) parent() PropertyPathItem {
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

func (i PropertyPath) parent() PropertyPathItem {
	return i[len(i)-1].parent()
}

func (i PropertyPath) Parts() []string {
	parts := make([]string, len(i))
	for idx, item := range i {
		switch item := item.(type) {
		case mapValuePathItem:
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

func (i PropertyPath) Last() PropertyPathItem {
	return i[len(i)-1]
}

type WalkPropertiesFunc func(path PropertyPath, err error) error

var stringerType = reflect.TypeOf((*fmt.Stringer)(nil)).Elem()

func mapKeys(m reflect.Value) ([]reflect.Value, error) {
	var toString func(elem reflect.Value) string
	keyType := m.Type().Key()
	switch {
	case keyType.Kind() == reflect.String:
		toString = func(elem reflect.Value) string { return elem.String() }

	case keyType.Implements(stringerType):
		toString = func(elem reflect.Value) string { return elem.Interface().(fmt.Stringer).String() }

	default:
		return nil, fmt.Errorf("expected map[string|fmt.Stringer]..., got %s", m.Type())
	}

	keys := m.MapKeys()
	sort.Slice(keys, func(i, j int) bool {
		a := toString(keys[i])
		b := toString(keys[j])
		return a < b
	})
	return keys, nil
}

var SkipProperty = fmt.Errorf("skip property")

// WalkProperties walks the properties of the resource, calling fn for each property. If fn returns
// SkipProperty, the property and its decendants (if a map or array type) is skipped. If fn returns
// StopWalk, the walk is stopped.
// NOTE: does not walk over the _keys_ of any maps, only values.
func (r *Resource) WalkProperties(fn WalkPropertiesFunc) error {
	queue := make([]PropertyPath, len(r.Properties))
	props := reflect.ValueOf(r.Properties)
	keys, _ := mapKeys(props)
	for i, k := range keys {
		queue[i] = PropertyPath{mapValuePathItem{m: props, key: k}}
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
			err = nil
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
				queue = append(queue, append(item, mapValuePathItem{
					_parent: item.Last(),
					m:       v,
					key:     k,
				}))
			}

		case reflect.Array, reflect.Slice:
			for i := 0; i < v.Len(); i++ {
				queue = append(queue, append(item, arrayIndexPathItem{
					_parent: item.Last(),
					a:       v,
					index:   i,
				}))
			}
		}
	}
	return err
}

func (p Properties) MarshalYAML() (interface{}, error) {
	// Is there a way to get the sorting for nested maps to work? This only does top-level.
	return yaml_util.MarshalMap(p, func(a, b string) bool { return a < b })
}
