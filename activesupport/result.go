package activesupport

import (
	"fmt"

	"github.com/pkg/errors"
)

// Result is a type that represents either success (Ok) or failure (Err).
type Result interface {
	Ok() interface{}
	Err() error

	IsOk() bool
	IsErr() bool

	And(Result) Result
	AndThen(op func(interface{}) Result) Result
	Or(Result) Result
	OrElse(op func(error) Result) Result

	Contains(val interface{}) bool

	Unwrap() interface{}

	Expect(msg string) interface{}
	ExpectErr(msg string) error
}

type SomeResult struct {
	val interface{}
	err error
}

func Return(val interface{}, err error) SomeResult {
	return SomeResult{val: val, err: err}
}

func Ok(val interface{}) SomeResult {
	return SomeResult{val: val}
}

func Err(err error) SomeResult {
	return SomeResult{err: err}
}

func ErrText(text string) SomeResult {
	return SomeResult{err: errors.New(text)}
}

// String returns a string representation of the self.
func (self SomeResult) String() string {
	if self.IsOk() {
		return fmt.Sprintf("Ok(%s)", self.val)
	}
	return fmt.Sprintf("Err(%s)", self.err)
}

// Err returns the error value.
func (self SomeResult) Err() error {
	return self.err
}

// Ok returns the success value.
func (self SomeResult) Ok() interface{} {
	return self.val
}

// IsOk returns true if the result is Ok.
func (self SomeResult) IsOk() bool {
	return self.err == nil
}

// IsErr returns true if the result if Err.
func (self SomeResult) IsErr() bool {
	return self.err != nil
}

// And returns res if the result is Ok, otherwise returns the Err value of self.
func (self SomeResult) And(res Result) Result {
	if res.Err() == nil && self.err == nil {
		return res
	}
	return self
}

// AndThen calls op if the result is Ok, otherwise returns the Err value of self.
func (self SomeResult) AndThen(op func(interface{}) Result) Result {
	if self.err == nil {
		return op(self.val)
	}
	return self
}

// Or returns res if the result is Err, otherwise returns the Ok value of self.
func (self SomeResult) Or(res Result) Result {
	if self.err != nil {
		return res
	}
	return self
}

// OrElse calls op if the result is Err, otherwise retuns the Ok value of self.
func (self SomeResult) OrElse(op func(error) Result) Result {
	if self.err != nil {
		return op(self.err)
	}
	return self
}

func (self SomeResult) Contains(val interface{}) bool {
	return self.val == val
}

func (self SomeResult) Expect(msg string) interface{} {
	if self.err != nil {
		panic(errors.WithMessage(self.err, msg))
	}
	return self.val
}

func (self SomeResult) ExpectErr(msg string) error {
	if self.err == nil {
		panic(errors.WithMessagef(errors.New(msg), "%v", self.val))
	}
	return self.err
}

func (self SomeResult) Unwrap() interface{} {
	if self.err != nil {
		panic(self.err)
	}
	return self.val
}
