package query

import (
	sitter "github.com/smacker/go-tree-sitter"
)

// Selector takes a MatchNodes and the program source, and optionally returns some value.
type Selector[T any] func(MatchNodes) (T, bool)

// Predicate takes some value and the program source, and returns a bool.
type Predicate[T any] interface {
	Test(input T) bool
}

func Select[T any](query NextFunc[MatchNodes], selector Selector[T]) NextFunc[T] {
	return SelectIf(query, selector, Is[MatchNodes](true))
}

func SelectIf[T any](query NextFunc[MatchNodes], selector Selector[T], predicate Predicate[MatchNodes]) NextFunc[T] {
	return func() (T, bool) {
		var zero T
		for {
			match, found := query()
			if !found {
				return zero, false
			}
			if !predicate.Test(match) {
				continue
			}
			if selected, found := selector(match); found {
				return selected, true
			}
		}
	}
}

func ContentOf(filter Selector[*sitter.Node]) Selector[string] {
	return func(match MatchNodes) (string, bool) {
		elem, found := filter(match)
		if !found {
			return "", false
		}
		elemContent := elem.Content()
		return elemContent, true
	}
}

// ParamNamed is a Selector that returns a param from the MatchNodes by name, if such a param exists.
func ParamNamed(paramName string) Selector[*sitter.Node] {
	return func(match MatchNodes) (*sitter.Node, bool) {
		if paramNode, found := match[paramName]; found {
			return paramNode, true
		} else {
			return nil, false
		}
	}
}

type Is[T any] bool

func (a Is[T]) Test(_ T) bool {
	return bool(a)
}

type Param struct {
	Named   string
	Matches Predicate[*sitter.Node]
}

func (wp Param) Test(nodes MatchNodes) bool {
	if paramNode, found := nodes[wp.Named]; found {
		return wp.Matches.Test(paramNode)
	}
	return false
}

type AllOf[T any] []Predicate[T]

func (a AllOf[T]) Test(input T) bool {
	for _, condition := range a {
		if !condition.Test(input) {
			return false
		}
	}
	return true
}

type HasContent string

func (hc HasContent) Test(node *sitter.Node) bool {
	return node.Content() == string(hc)
}

type HasParent struct {
	With Predicate[*sitter.Node]
}

func (hp HasParent) Test(node *sitter.Node) bool {
	parent := node.Parent()
	if parent == nil {
		return false
	}
	return hp.With.Test(parent)
}

type Type string

func (ot Type) Test(node *sitter.Node, _ []byte) bool {
	return node.Type() == string(ot)
}
