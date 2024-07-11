package operational_eval

import (
	"errors"
	"fmt"
)

type EnqueueErrors map[Key]error

func (e EnqueueErrors) Error() string {
	return fmt.Sprintf("enqueue errors: %v", map[Key]error(e))
}

func (e EnqueueErrors) Unwrap() []error {
	errs := make([]error, 0, len(e))
	for k, err := range e {
		errs = append(errs, fmt.Errorf("%s: %w", k, err))
	}
	return errs
}

func (e *EnqueueErrors) Append(key Key, err error) {
	if err == nil {
		return
	}
	if *e == nil {
		*e = make(EnqueueErrors)
	}
	if x, ok := (*e)[key]; ok {
		err = errors.Join(x, err)
	}
	(*e)[key] = err
}
