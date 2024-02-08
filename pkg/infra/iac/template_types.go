package iac

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"

	construct "github.com/klothoplatform/klotho/pkg/construct"
)

// templateString is a quoted string, but evaluates as false-y in a template if the string is empty.
//
// Example:
//
//	data.Value = templateString("")
//
//	{{ if .Value }}value: {{ .Value }}{{ else }}no value: {{ .Value }}{{ end }}
//
// results in:
//
//	no value: ""
type templateString string

func (s templateString) String() string {
	return strconv.Quote(string(s))
}

type (
	appliedOutput struct {
		Ref  string
		Name string
	}

	appliedOutputs []appliedOutput
)

func (tc TemplatesCompiler) NewAppliedOutput(ref construct.PropertyRef, name string) appliedOutput {
	ao := appliedOutput{Name: name}
	if ao.Name == "" {
		ao.Name = tc.vars[ref.Resource]
	}
	if ref.Property == "" {
		ao.Ref = tc.vars[ref.Resource]
	} else {
		ao.Ref = fmt.Sprintf("%s.%s", tc.vars[ref.Resource], ref.Property)
	}
	return ao
}

func (ao *appliedOutputs) dedupe() error {
	if ao == nil || len(*ao) == 0 {
		return nil
	}

	var err error
	values := make(map[appliedOutput]struct{})
	names := make(map[string]struct{})
	for i := 0; i < len(*ao); i++ {
		v := (*ao)[i]
		if _, ok := values[v]; ok {
			i--
			// Delete the duplicate (shift everything down)
			copy((*ao)[i:], (*ao)[i+1:])
			*ao = (*ao)[:len(*ao)-1]
			continue
		}
		values[v] = struct{}{}
		if _, ok := names[v.Name]; ok {
			err = errors.Join(err, fmt.Errorf("duplicate applied output name %q", v.Name))
		}
		names[v.Name] = struct{}{}
	}
	sort.Sort(*ao)
	return err
}

func (ao appliedOutputs) Len() int {
	return len(ao)
}

func (ao appliedOutputs) Less(i, j int) bool {
	if ao[i].Ref < ao[j].Ref {
		return true
	}

	return ao[i].Name < ao[j].Name
}

func (ao appliedOutputs) Swap(i, j int) {
	ao[i], ao[j] = ao[j], ao[i]
}

// Render writes the applied outputs to the given writer, running the given function in between
// as the body of the apply function.
func (ao appliedOutputs) Render(out io.Writer, f func(io.Writer) error) error {
	var errs error
	write := func(msg string, args ...interface{}) {
		_, err := fmt.Fprintf(out, msg, args...)
		errs = errors.Join(errs, err)
	}
	switch len(ao) {
	case 0:
		return nil

	case 1:
		write("%s.apply(%s => { return ",
			ao[0].Ref,
			ao[0].Name,
		)

	default:
		write("pulumi.all([")
		for i := 0; i < len(ao); i++ {
			write(ao[i].Ref)
			if i < len(ao)-1 {
				write(", ")
			}
		}
		write("])\n.apply(([")
		for i := 0; i < len(ao); i++ {
			write(ao[i].Name)
			if i < len(ao)-1 {
				write(", ")
			}
		}
		write("]) => {\n    return ")
	}

	errs = errors.Join(errs, f(out))
	write("\n})")
	return errs
}

// jsonValue is a value that will be marshaled to JSON when evaluated in a template.
// But also lets the value to be used as-is in template functions (such as map access or number comparisons).
//
// Unfortunately, we can't use the same trick as in [templateString] because you can't define methods on an interface,
// which
//
//	type jsonValue any
//
// would be an interface. This means that the value won't be false-y and will have to access the underlying type
// via `.Raw` in the template.
type jsonValue struct {
	Raw any
}

func (j jsonValue) String() string {
	b, err := json.Marshal(j.Raw)
	if err != nil {
		// pretty unlikely to happen, but if it does, the template evaluation (via fmt.Fprint)
		// with recover this panic.
		// If at some point, text/template could support MarshalText (which can return an error)
		// we should migrate to using that.
		panic(err)
	}
	return string(b)
}
