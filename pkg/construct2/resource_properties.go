package construct2

import (
	"fmt"
	"strconv"
	"strings"
)

type (
	Properties = map[string]interface{}
)

func (r *Resource) SetProperty(path string, value any) error {
	parts := splitPath(path)
	if len(parts) == 0 {
		return fmt.Errorf("empty path")
	}
	on, err := r.propertyValue(parts[:len(parts)-1])
	if err != nil {
		return err
	}
	last := parts[len(parts)-1]
	switch on := on.(type) {
	case map[string]any:
		on[last] = value

	case []any:
		idxStr := last[1 : len(last)-1]
		idx, err := strconv.Atoi(idxStr)
		if err != nil {
			return &PropertyTypeError{Path: parts, Cause: err}
		}
		if idx < 0 || idx >= len(on) {
			return &PropertyTypeError{
				Path:  parts,
				Cause: fmt.Errorf("array index out of bounds: %d (length %d)", idx, len(on)),
			}
		}
		on[idx] = value

	default:
		return &PropertyTypeError{
			Path:  parts,
			Cause: fmt.Errorf("expected map or array, got %T", on),
		}
	}
	return nil
}

func (r *Resource) AppendProperty(path string, value any) error {
	return nil
}

func (r *Resource) GetProperty(path string) (any, error) {
	return r.propertyValue(splitPath(path))
}

func (r *Resource) RemoveProperty(path string) error {
	return nil
}

type (
	propertyPathItem interface {
		Set(value any) error
		Append(value any) error
		Remove(value any) error
		Get() any
	}

	propertyPath []propertyPathItem

	scalarPathItem struct {
		scalar any
	}

	mapKeyPathItem struct {
		m   map[string]any
		key string
	}

	arrayIndexPathItem struct {
		parent propertyPathItem
		a      []any
		index  int
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

// func makePropertyPath(props Properties, path string) (propertyPath, error) {
// 	value = any(props)
// }

func (r *Resource) propertyValue(pathParts []string) (any, error) {
	value := any(r.Properties)
	for i, part := range pathParts {
		switch part[0] {
		case '.':
			part = part[1:]
			fallthrough
		default:
			m, ok := value.(map[string]any)
			if !ok {
				return nil, &PropertyTypeError{
					Path:  pathParts[:i-1],
					Cause: fmt.Errorf("expected map, got %T", value),
				}
			}
			value = m[part]

		case '[':
			m, ok := value.([]any)
			if !ok {
				return nil, &PropertyTypeError{
					Path:  pathParts[:i-1],
					Cause: fmt.Errorf("expected array, got %T", value),
				}
			}
			if len(part) < 2 || part[len(part)-1] != ']' {
				return nil, &PropertyTypeError{
					Path:  pathParts[:i],
					Cause: fmt.Errorf("invalid array index format, got %q", part),
				}
			}
			idxStr := part[1 : len(part)-1]
			idx, err := strconv.Atoi(idxStr)
			if err != nil {
				return nil, &PropertyTypeError{Path: pathParts[:i], Cause: err}
			}
			if idx < 0 || idx >= len(m) {
				return nil, &PropertyTypeError{
					Path:  pathParts[:i],
					Cause: fmt.Errorf("array index out of bounds: %d (length %d)", idx, len(m)),
				}
			}
			value = m[idx]
		}
	}
	return value, nil
}

type PropertyTypeError struct {
	Path  []string
	Cause error
}

func (e *PropertyTypeError) Error() string {
	return fmt.Sprintf("error in path %s: %v",
		strings.Join(e.Path, ""),
		e.Cause,
	)
}

func (e *PropertyTypeError) Unwrap() error {
	return e.Cause
}
