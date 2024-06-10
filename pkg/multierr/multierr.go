package multierr

import (
	"bytes"
	"errors"
	"fmt"
)

type Error []error

func (e Error) Error() string {
	switch len(e) {
	case 0:
		// Generally won't be called, but here for completion sake
		return "<nil>"

	case 1:
		return e[0].Error()

	default:
		buf := new(bytes.Buffer)
		fmt.Fprintf(buf, "%d errors occurred:", len(e))
		for _, err := range e {
			fmt.Fprintf(buf, `
	* %v`, err)
		}
		return buf.String()
	}
}

// Append will mutate e and append the error. Will no-op if `err == nil`.
// Typical usage should be via auto-referencing [syntax sugar](https://go.dev/ref/spec#Calls):
//
//	var e Error
//	e.Append(err)
func (e *Error) Append(err error) {
	switch {
	case e == nil:
		// if the pointer to the array is nil, nothing we can do.
		// this shouldn't normally happen unless callers are for some reason
		// using `*Error`, which they shouldn't (`Error` as an array is already a pointer)

	case err == nil:
		// Do nothing

	case *e == nil:
		*e = Error{err}

	default:
		*e = append(*e, err)
	}
}

// Append adds err2 to err1.
// - If err1 and err2 are nil, returns nil
// - If err1 is nil, returns an [Error] with only err2
// - If err2 is nil, returns an [Error] with only err1
// - If err1 is an [Error], it returns a copy with err2 appended
// - Otherwise, returns a new [Error] with err1 and err2 as the sole elements
// NOTE: unlike `err1.Append`, this does not mutate err1.
func Append(err1, err2 error) Error {
	switch {
	case err1 == nil && err2 == nil:
		return nil

	case err1 == nil:
		return Error{err2}

	case err2 == nil:
		return Error{err1}
	}

	if merr, ok := err1.(Error); ok {
		merr.Append(err2)
		return merr
	}
	return Error{err1, err2}
}

// ErrOrNil is used to convert this multierr into a [error]. This is necessary because it is a typed nil
//
//	func example() error {
//		var e Error
//		return e
//	}
//	if example() != nil {
//		! this will run!
//	}
//
// in otherwords,
//
//	(Error)(nil) != nil
//
// Additionally, if there's only a single error, it will automatically unwrap it.
func (e Error) ErrOrNil() error {
	switch len(e) {
	case 0:
		return nil

	case 1:
		return e[0]

	default:
		return e
	}
}

// Unwrap implements the interface used in [errors.Unwrap]
func (e Error) Unwrap() error {
	switch len(e) {
	case 0:
		return nil

	case 1:
		return e[0]

	default:
		return e[1:]
	}
}

// As implements the interface used in [errors.As] by iterating through the members
// returning true on the first match.
func (e Error) As(target interface{}) bool {
	for _, err := range e {
		if errors.As(err, target) {
			return true
		}
	}
	return false
}

// Is implements the interface used in [errors.Is] by iterating through the members
// returning true on the first match.
func (e Error) Is(target error) bool {
	for _, err := range e {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}
