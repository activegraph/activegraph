package activesupport

import (
	"fmt"

	"github.com/pkg/errors"
)

// Result is a type that represents either success (Ok) or failure (Err).
type result interface {
	Ok() Option
	Err() error

	IsOk() bool
	IsErr() bool

	And(Result) Result
	AndThen(op func(interface{}) Result) Result
	Or(Result) Result
	OrElse(op func(error) Result) Result

	Contains(val interface{}) bool

	Unwrap() interface{}
	UnwrapOr(val interface{}) interface{}

	Expect(msg string) interface{}
	ExpectErr(msg string) error
}

var _ result = Result{}

type Result struct {
	t   T
	val interface{}
	err error
}

func Return(t T, val interface{}, err error) Result {
	return Result{t: t, val: val, err: err}
}

func Ok(t T, val interface{}) Result {
	return Result{t: t, val: val}
}

func Err(t T, err error) Result {
	return Result{t: t, err: err}
}

func ErrText(t T, text string) Result {
	return Result{t: t, err: errors.New(text)}
}

func (self Result) T() T {
	return self.t
}

// String returns a string representation of the self.
func (self Result) String() string {
	if self.IsOk() {
		return fmt.Sprintf("Ok(%s)", self.val)
	}
	return fmt.Sprintf("Err(%T)", self.err)
}

// Err returns the error value.
func (self Result) Err() error {
	return self.err
}

// Ok returns the success value.
func (self Result) Ok() Option {
	if self.IsErr() {
		return None(self.T())
	}
	return Some(self.T(), self.val)
}

// IsOk returns true if the result is Ok.
func (self Result) IsOk() bool {
	return self.err == nil
}

// IsErr returns true if the result if Err.
func (self Result) IsErr() bool {
	return self.err != nil
}

// And returns res if the result is Ok, otherwise returns the Err value of self.
func (self Result) And(res Result) Result {
	if res.Err() == nil && self.err == nil {
		return res
	}
	return self
}

// AndThen calls op if the result is Ok, otherwise returns the Err value of self.
func (self Result) AndThen(op func(interface{}) Result) Result {
	if self.err == nil {
		return op(self.val)
	}
	return self
}

// Or returns res if the result is Err, otherwise returns the Ok value of self.
func (self Result) Or(res Result) Result {
	if self.err != nil {
		return res
	}
	return self
}

// OrElse calls op if the result is Err, otherwise retuns the Ok value of self.
func (self Result) OrElse(op func(error) Result) Result {
	if self.err != nil {
		return op(self.err)
	}
	return self
}

func (self Result) Contains(val interface{}) bool {
	return self.val == val
}

func (self Result) Expect(msg string) interface{} {
	if self.err != nil {
		panic(errors.WithMessage(self.err, msg))
	}
	return self.val
}

func (self Result) ExpectErr(msg string) error {
	if self.err == nil {
		panic(errors.WithMessagef(errors.New(msg), "%v", self.val))
	}
	return self.err
}

func (self Result) Unwrap() interface{} {
	if self.err != nil {
		panic(self.err)
	}
	return self.val
}

// UnwrapOr returns the contained Ok value or a provided default.
func (self Result) UnwrapOr(val interface{}) interface{} {
	if self.IsErr() {
		return val
	}
	return self.val
}
