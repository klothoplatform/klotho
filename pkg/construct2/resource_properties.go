package construct2

import (
	"fmt"
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
		m       map[string]any
		key     string
	}

	arrayIndexPathItem struct {
		_parent propertyPathItem
		a       []any
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
	value := any(r.Properties)
	for i, part := range pathParts {
		switch part[0] {
		case '.':
			part = part[1:]
			fallthrough
		default:
			m, ok := value.(map[string]any)
			if !ok {
				return nil, &PropertyPathError{
					Path:  pathParts[:i-1],
					Cause: fmt.Errorf("expected map, got %T", value),
				}
			}
			if i > 0 {
				path[i] = mapKeyPathItem{_parent: path[i-1], m: m, key: part}
			} else {
				path[i] = mapKeyPathItem{m: m, key: part}
			}
			value = m[part]

		case '[':
			m, ok := value.([]any)
			if !ok {
				return nil, &PropertyPathError{
					Path:  pathParts[:i-1],
					Cause: fmt.Errorf("expected array, got %T", value),
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
			if idx < 0 || idx >= len(m) {
				return nil, &PropertyPathError{
					Path:  pathParts[:i],
					Cause: fmt.Errorf("array index out of bounds: %d (length %d)", idx, len(m)),
				}
			}
			path[i] = arrayIndexPathItem{_parent: path[i-1], a: m, index: idx}
			value = m[idx]
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

func (i mapKeyPathItem) Set(value any) error {
	i.m[i.key] = value
	return nil
}

func (i mapKeyPathItem) Append(value any) error {
	a, ok := i.m[i.key].([]any)
	if !ok {
		return &PropertyPathError{
			Path:  itemToPath(i),
			Cause: fmt.Errorf("expected array for append, got %T", i.m[i.key]),
		}
	}
	i.m[i.key] = append(a, value)
	return nil
}

func arrRemoveByValue(arr []any, value any) ([]any, error) {
	newArr := make([]any, 0, len(arr))
	for _, item := range arr {
		if item != value {
			newArr = append(newArr, item)
		}
	}
	if len(newArr) == len(arr) {
		return nil, fmt.Errorf("value %v not found in array", value)
	}
	return newArr, nil
}

func (i mapKeyPathItem) Remove(value any) error {
	if value == nil {
		delete(i.m, i.key)
		return nil
	}
	arr, ok := i.m[i.key].([]any)
	if !ok {
		return &PropertyPathError{
			Path:  itemToPath(i),
			Cause: fmt.Errorf("for non-nil value'd (%v), must be array (got %T) to remove by value", value, i.m[i.key]),
		}
	}
	newArr, err := arrRemoveByValue(arr, value)
	if err != nil {
		return &PropertyPathError{Path: itemToPath(i), Cause: err}
	}
	i.m[i.key] = newArr
	return nil
}

func (i mapKeyPathItem) Get() any {
	return i.m[i.key]
}

func (i mapKeyPathItem) parent() propertyPathItem {
	return i._parent
}

func (i arrayIndexPathItem) Set(value any) error {
	i.a[i.index] = value
	return nil
}

func (i arrayIndexPathItem) Append(value any) error {
	a, ok := i.a[i.index].([]any)
	if !ok {
		return &PropertyPathError{
			Path:  itemToPath(i),
			Cause: fmt.Errorf("expected array for append, got %T", i.a[i.index]),
		}
	}
	i.a[i.index] = append(a, value)
	return nil
}

func (i arrayIndexPathItem) Remove(value any) error {
	if value == nil {
		i.a = append(i.a[:i.index], i.a[i.index+1:]...)
		return i._parent.Set(i.a)
	}

	arr, ok := i.a[i.index].([]any)
	if !ok {
		return &PropertyPathError{
			Path:  itemToPath(i),
			Cause: fmt.Errorf("for non-nil value'd (%v), must be array (got %T) to remove by value", value, i.a[i.index]),
		}
	}
	newArr, err := arrRemoveByValue(arr, value)
	if err != nil {
		return &PropertyPathError{Path: itemToPath(i), Cause: err}
	}
	i.a[i.index] = newArr
	return nil
}

func (i arrayIndexPathItem) Get() any {
	return i.a[i.index]
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
			key := item.key
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
