package yaml_util

import (
	"errors"
	"fmt"
	"sort"

	"gopkg.in/yaml.v3"
)

var nullNode = &yaml.Node{
	Kind:  yaml.ScalarNode,
	Tag:   "!!null",
	Value: "",
}

func MarshalMap[K comparable, V any](m map[K]V, less func(K, K) bool) (*yaml.Node, error) {
	if len(m) == 0 {
		return nullNode, nil
	}

	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(
		keys,
		func(i, j int) bool { return less(keys[i], keys[j]) },
	)

	node := &yaml.Node{Kind: yaml.MappingNode}
	var errs error
	for _, k := range keys {
		var v any = m[k]
		var keyNode yaml.Node
		if err := keyNode.Encode(k); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to encode key %v: %w", k, err))
			continue
		}
		var valueNode yaml.Node
		switch v := v.(type) {
		case *yaml.Node:
			valueNode = *v

		default:
			if err := valueNode.Encode(m[k]); err != nil {
				errs = errors.Join(errs, fmt.Errorf("failed to encode value for %v: %w", k, err))
				continue
			}
		}
		node.Content = append(
			node.Content,
			&keyNode,
			&valueNode,
		)
	}

	return node, nil
}
