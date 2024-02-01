package engine_errs

import (
	"fmt"
	"strings"
)

type (
	EngineError interface {
		error
		// ToJSONMap returns a map that can be marshaled to JSON. Uses this instead of MarshalJSON to avoid
		// repeated marshalling of common fields (such as 'error_code') and to allow for consistent formatting
		// (eg for pretty-print).
		ToJSONMap() map[string]any
		ErrorCode() ErrorCode
	}

	ErrorCode string

	InternalError struct {
		Err error
	}

	ErrorTree struct {
		Chain    []string    `json:"chain,omitempty"`
		Children []ErrorTree `json:"children,omitempty"`
	}
)

const (
	InternalErrCode     ErrorCode = "internal"
	ConfigInvalidCode   ErrorCode = "config_invalid"
	EdgeInvalidCode     ErrorCode = "edge_invalid"
	EdgeUnsupportedCode ErrorCode = "edge_unsupported"
)

func (e InternalError) Error() string {
	return fmt.Sprintf("internal error: %v", e.Err)
}

func (e InternalError) ErrorCode() ErrorCode {
	return InternalErrCode
}

func (e InternalError) ToJSONMap() map[string]any {
	return map[string]any{}
}

func (e InternalError) Unwrap() error {
	return e.Err
}

type (
	chainErr interface {
		error
		Unwrap() error
	}
	joinErr interface {
		error
		Unwrap() []error
	}
)

func unwrapChain(err error) (chain []string, last joinErr) {
	for current := err; current != nil; {
		var next error
		cc, ok := current.(chainErr)
		if ok {
			next = cc.Unwrap()
		} else {
			joined, ok := current.(joinErr)
			if ok {
				jerrs := joined.Unwrap()
				if len(jerrs) == 1 {
					next = jerrs[0]
				} else {
					last = joined
					return
				}
			} else {
				chain = append(chain, current.Error())
				return
			}
		}
		msg := strings.TrimSuffix(strings.TrimSuffix(current.Error(), next.Error()), ": ")
		if msg != "" {
			chain = append(chain, msg)
		}
		current = next
	}
	return
}

func ErrorsToTree(err error) (tree ErrorTree) {
	if err == nil {
		return
	}
	if t, ok := err.(ErrorTree); ok {
		return t
	}

	var joined joinErr
	tree.Chain, joined = unwrapChain(err)

	if joined != nil {
		errs := joined.Unwrap()
		tree.Children = make([]ErrorTree, len(errs))
		for i, e := range errs {
			tree.Children[i] = ErrorsToTree(e)
		}
	}
	return
}

func (t ErrorTree) Error() string {
	sb := &strings.Builder{}
	t.print(sb, 0, 0)
	return sb.String()
}

func (t ErrorTree) print(out *strings.Builder, indent int, childChar rune) {
	prefix := strings.Repeat("\t", indent)
	delim := ""
	if childChar != 0 {
		delim = string(childChar) + " "
	}
	fmt.Fprintf(out, "%s%s%v\n", prefix, delim, t.Chain)
	for i, child := range t.Children {
		char := '├'
		if i == len(t.Children)-1 {
			char = '└'
		}
		child.print(out, indent+1, char)
	}
}
