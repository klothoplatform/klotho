package engine_errs

import (
	"fmt"

	construct "github.com/klothoplatform/klotho/pkg/construct"
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
)

const (
	InternalErrCode     ErrorCode = "internal"
	ConfigInvalidCode   ErrorCode = "config_invalid"
	EdgeInvalidCode     ErrorCode = "edge_invalid"
	EdgeUnsupportedCode ErrorCode = "edge_unsupported"
)

type InternalError struct {
	Err error
}

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

type UnsupportedExpansionErr struct {
	// ExpandEdge is the overall edge that is being expanded
	ExpandEdge construct.SimpleEdge
	// SatisfactionEdge is the specific edge that was being expanded when the error occurred
	SatisfactionEdge construct.SimpleEdge
	Classification   string
}

func (e UnsupportedExpansionErr) Error() string {
	if e.SatisfactionEdge.Source.IsZero() || e.ExpandEdge == e.SatisfactionEdge {
		return fmt.Sprintf("unsupported expansion %s in %s", e.ExpandEdge, e.Classification)
	}
	return fmt.Sprintf(
		"while expanding %s, unsupported expansion of %s in %s",
		e.ExpandEdge,
		e.Classification,
		e.SatisfactionEdge,
	)
}

func (e UnsupportedExpansionErr) ErrorCode() ErrorCode {
	return EdgeUnsupportedCode
}

func (e UnsupportedExpansionErr) ToJSONMap() map[string]any {
	m := map[string]any{
		"satisfaction_edge": e.SatisfactionEdge,
	}
	if !e.ExpandEdge.Source.IsZero() {
		m["expand_edge"] = e.ExpandEdge
	}
	if e.Classification != "" {
		m["classification"] = e.Classification
	}
	return m
}

type InvalidPathErr struct {
	// ExpandEdge is the overall edge that is being expanded
	ExpandEdge construct.SimpleEdge
	// SatisfactionEdge is the specific edge that was being expanded when the error occurred
	SatisfactionEdge construct.SimpleEdge
	Classification   string
}

func (e InvalidPathErr) Error() string {
	if e.SatisfactionEdge.Source.IsZero() || e.ExpandEdge == e.SatisfactionEdge {
		return fmt.Sprintf("invalid expansion %s in %s", e.ExpandEdge, e.Classification)
	}
	return fmt.Sprintf(
		"while expanding %s, invalid expansion of %s in %s",
		e.ExpandEdge,
		e.Classification,
		e.SatisfactionEdge,
	)
}

func (e InvalidPathErr) ErrorCode() ErrorCode {
	return EdgeInvalidCode
}

func (e InvalidPathErr) ToJSONMap() map[string]any {
	m := map[string]any{
		"satisfaction_edge": e.SatisfactionEdge,
	}
	if !e.ExpandEdge.Source.IsZero() {
		m["expand_edge"] = e.ExpandEdge
	}
	if e.Classification != "" {
		m["classification"] = e.Classification
	}
	return m
}
