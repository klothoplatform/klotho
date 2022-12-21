package multierr

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestError_Error(t *testing.T) {
	// helper to create optional string `wantEqual`
	ref := func(s string) *string { return &s }

	tests := []struct {
		name         string
		errs         []error
		wantEqual    *string
		wantContains []string
	}{
		{
			name:      "empty",
			wantEqual: ref("<nil>"),
		},
		{
			name:      "single Err",
			errs:      []error{errors.New("test error")},
			wantEqual: ref("test error"),
		},
		{
			name: "multi error",
			errs: []error{errors.New("error A"), errors.New("error B")},
			wantContains: []string{
				"2 errors",
				"error A",
				"error B",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			e := Error(tt.errs)
			if tt.wantEqual != nil {
				assert.Equal(*tt.wantEqual, e.Error())
			} else {
				msg := e.Error()
				for _, contains := range tt.wantContains {
					assert.Contains(msg, contains)
				}
			}
		})
	}
}

func TestError_Append(t *testing.T) {
	tests := []struct {
		name string
		e    Error
		add  error
	}{
		{
			name: "append simple",
			e:    Error{errors.New("a")},
			add:  errors.New("b"),
		},
		{
			name: "append to nil",
			e:    nil,
			add:  errors.New("a"),
		},
		{
			name: "append nil err",
			e:    Error{errors.New("a")},
			add:  nil,
		},
		{
			name: "append all nil",
			e:    nil,
			add:  nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			beforeLen := len(tt.e)
			tt.e.Append(tt.add)
			if tt.add != nil {
				assert.ErrorIs(tt.e, tt.add)
			} else {
				assert.Equal(beforeLen, len(tt.e))
			}
		})
	}
}

func TestAppend(t *testing.T) {
	tests := []struct {
		name string
		err1 error
		err2 error
	}{
		{
			name: "simple append",
			err1: errString("a"),
			err2: errString("b"),
		},
		{
			name: "append to multierr",
			err1: Error{errString("a")},
			err2: errString("b"),
		},
		{
			name: "nil err1",
			err1: nil,
			err2: errString("b"),
		},
		{
			name: "nil err2",
			err1: errString("a"),
			err2: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			got := Append(tt.err1, tt.err2)
			if merr, ok := tt.err1.(Error); ok {
				assert.Len(got, len(merr)+1)
				assert.Equal(tt.err2, got[len(got)-1])
			} else if tt.err1 != nil && tt.err2 != nil && assert.Len(got, 2) {
				assert.Equal(got[0], tt.err1)
				assert.Equal(got[1], tt.err2)
			}
		})
	}
}

func TestError_ErrOrNil(t *testing.T) {
	singleErr := errors.New("a")
	nonEmptyList := Error{errors.New("a"), errors.New("b")}
	tests := []struct {
		name string
		e    Error
		want error
	}{
		{
			name: "nil is nil",
			e:    nil,
			want: nil,
		},
		{
			name: "empty is nil",
			e:    Error{},
			want: nil,
		},
		{
			name: "single err is unwrapped",
			e:    Error{singleErr},
			want: singleErr,
		},
		{
			name: "multierror stays as-is",
			e:    nonEmptyList,
			want: nonEmptyList,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			assert.Equal(tt.want, tt.e.ErrOrNil())
		})
	}
}

func TestError_Unwrap(t *testing.T) {
	a := errors.New("a")
	errList := Error{
		a,
		errors.New("b"),
		errors.New("c"),
	}
	tests := []struct {
		name string
		e    Error
		want error
	}{
		{
			name: "empty",
			e:    Error{},
			want: nil,
		},
		{
			name: "nil",
			e:    nil,
			want: nil,
		},
		{
			name: "single",
			e:    Error{a},
			want: a,
		},
		{
			name: "multi",
			e:    errList,
			want: errList[1:],
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			assert.Equal(tt.want, tt.e.Unwrap())
		})
	}
}

type errString string

func (s errString) Error() string { return string(s) }

func TestError_As(t *testing.T) {
	// Can't use table-based tests for these since they use reflect and the field type
	// of `error` wouldn't convey the correct type. Can't have an array of heterogeneous
	// generic types either (for a `[T error]` generic test struct).

	t.Run("simple as", func(t *testing.T) {
		assert := assert.New(t)
		e := Error{errString("a")}
		var s errString
		assert.ErrorAs(e, &s)
	})

	t.Run("as not match", func(t *testing.T) {
		assert := assert.New(t)
		e := Error{errors.New("a")}
		var s errString
		assert.False(e.As(&s))
	})

	t.Run("as match second", func(t *testing.T) {
		assert := assert.New(t)
		e := Error{errors.New("a"), errString("b")}
		var s errString
		assert.ErrorAs(e, &s)
	})

	t.Run("nil not as", func(t *testing.T) {
		assert := assert.New(t)
		var e Error
		var s errString
		assert.False(e.As(&s))
	})

	t.Run("sub-multi as", func(t *testing.T) {
		// This test demonstrates why As needs to be implemented.
		// When As is removed, this test fails.
		assert := assert.New(t)
		e := Error{Error{errString("b")}, errors.New("a")}
		var s errString
		assert.ErrorAs(e, &s)
	})
}

func TestError_Is(t *testing.T) {
	// Can't use table-based tests for these since they use reflect and the field type
	// of `error` wouldn't convey the correct type. Can't have an array of heterogeneous
	// generic types either (for a `[T error]` generic test struct).

	t.Run("simple is", func(t *testing.T) {
		assert := assert.New(t)

		single := errString("a")
		e := Error{single}
		assert.ErrorIs(e, single)
	})

	t.Run("simple is not", func(t *testing.T) {
		assert := assert.New(t)

		e := Error{errString("a")}
		assert.NotErrorIs(e, errString("b"))
	})

	t.Run("is match second", func(t *testing.T) {
		assert := assert.New(t)

		single := errString("b")
		e := Error{errString("a"), single}
		assert.ErrorIs(e, single)
	})

	t.Run("nil not is", func(t *testing.T) {
		assert := assert.New(t)
		var e Error
		assert.NotErrorIs(e, errString("a"))
	})

	t.Run("sub-multi is", func(t *testing.T) {
		// This test demonstrates why Is needs to be implemented.
		// When Is is removed, this test fails.
		assert := assert.New(t)
		single := errString("a")
		e := Error{Error{single}, errors.New("a")}
		assert.ErrorIs(e, single)
	})
}
