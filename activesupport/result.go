package activesupport

import (
	"fmt"
)

// Result is a type that represents either success (Ok) or failure (Err).
type Result[T comparable] interface {
	Ok() Option[T]
	Err() error

	IsOk() bool
	IsErr() bool

	And(Result[T]) Result[T]
	AndThen(op func(T) Result[T]) Result[T]
	Or(Result[T]) Result[T]
	OrElse(op func(error) Result[T]) Result[T]

	Contains(val T) bool

	Unwrap() T
	UnwrapOr(val T) T

	Expect(msg string) T
	ExpectErr(msg string) error
}

type AnyResult[T comparable] struct {
	val T
	err error
}

func Return[T comparable](val T, err error) AnyResult[T] {
	return AnyResult[T]{val: val, err: err}
}

func Ok[T comparable](val T) AnyResult[T] {
	return AnyResult[T]{val: val}
}

func Err[T comparable](err error) AnyResult[T] {
	return AnyResult[T]{err: err}
}

func ErrText[T comparable](text string) AnyResult[T] {
	return AnyResult[T]{err: fmt.Errorf(text)}
}

// String returns a string representation of the self.
func (self AnyResult[T]) String() string {
	if self.IsOk() {
		return fmt.Sprintf("Ok(%v)", self.val)
	}
	return fmt.Sprintf("Err(%v)", self.err)
}

// Err returns the error value.
func (self AnyResult[T]) Err() error {
	return self.err
}

// Ok returns the success value.
func (self AnyResult[T]) Ok() Option[T] {
	if self.IsErr() {
		return None[T]()
	}
	return Some(self.val)
}

// IsOk returns true if the result is Ok.
func (self AnyResult[T]) IsOk() bool {
	return self.err == nil
}

// IsErr returns true if the result if Err.
func (self AnyResult[T]) IsErr() bool {
	return self.err != nil
}

// And returns res if the result is Ok, otherwise returns the Err value of self.
func (self AnyResult[T]) And(res Result[T]) Result[T] {
	if res.Err() == nil && self.err == nil {
		return res
	}
	return self
}

// AndThen calls op if the result is Ok, otherwise returns the Err value of self.
func (self AnyResult[T]) AndThen(op func(T) Result[T]) Result[T] {
	if self.err == nil {
		return op(self.val)
	}
	return self
}

// Or returns res if the result is Err, otherwise returns the Ok value of self.
func (self AnyResult[T]) Or(res Result[T]) Result[T] {
	if self.err != nil {
		return res
	}
	return self
}

// OrElse calls op if the result is Err, otherwise retuns the Ok value of self.
func (self AnyResult[T]) OrElse(op func(error) Result[T]) Result[T] {
	if self.err != nil {
		return op(self.err)
	}
	return self
}

func (self AnyResult[T]) Contains(val T) bool {
	return self.val == val
}

func (self AnyResult[T]) Expect(msg string) T {
	if self.err != nil {
		panic(fmt.Errorf("%s: %w", msg, self.err))
	}
	return self.val
}

func (self AnyResult[T]) ExpectErr(msg string) error {
	if self.err == nil {
		panic(fmt.Errorf("%v: %s", self.val, msg))
	}
	return self.err
}

func (self AnyResult[T]) Unwrap() T {
	if self.err != nil {
		panic(self.err)
	}
	return self.val
}

// UnwrapOr returns the contained Ok value or a provided default.
func (self AnyResult[T]) UnwrapOr(val T) T {
	if self.IsErr() {
		return val
	}
	return self.val
}

type FutureResult[T comparable] struct {
	callstack func() Result[T]
	computed  *Result[T]
}

func FutureOk[T comparable](val T) FutureResult[T] {
	return FutureResult[T]{callstack: func() Result[T] {
		return Ok(val)
	}}
}

func FutureErr[T comparable](err error) FutureResult[T] {
	return FutureResult[T]{callstack: func() Result[T] {
		return Err[T](err)
	}}
}

func (self FutureResult[T]) compute() Result[T] {
	if self.computed == nil {
		computed := self.callstack()
		self.computed = &computed
	}
	return *self.computed
}

func (self FutureResult[T]) push(op func(Result[T]) Result[T]) Result[T] {
	// Erase previous computations as new operations will be stacked.
	self.computed = nil
	parentOp := self.callstack

	self.callstack = func() Result[T] {
		return op(parentOp())
	}
	return self
}

func (self FutureResult[T]) Ok() Option[T] {
	return self.compute().Ok()
}

func (self FutureResult[T]) Err() error {
	return self.compute().Err()
}

func (self FutureResult[T]) IsOk() bool {
	return self.compute().IsOk()
}

func (self FutureResult[T]) IsErr() bool {
	return self.compute().IsErr()
}

func (self FutureResult[T]) And(res Result[T]) Result[T] {
	return self.compute().And(res)
}

func (self FutureResult[T]) AndThen(op func(T) Result[T]) Result[T] {
	return self.push(func(r Result[T]) Result[T] { return r.AndThen(op) })
}

func (self FutureResult[T]) Or(res Result[T]) Result[T] {
	return self.compute().Or(res)
}

func (self FutureResult[T]) OrElse(op func(error) Result[T]) Result[T] {
	return self.push(func(r Result[T]) Result[T] { return r.OrElse(op) })
}

func (self FutureResult[T]) Contains(val T) bool {
	return self.compute().Contains(val)
}

func (self FutureResult[T]) Unwrap() T {
	return self.compute().Unwrap()
}

func (self FutureResult[T]) UnwrapOr(val T) T {
	return self.compute().UnwrapOr(val)
}

func (self FutureResult[T]) Expect(msg string) T {
	return self.compute().Expect(msg)
}

func (self FutureResult[T]) ExpectErr(msg string) error {
	return self.compute().ExpectErr(msg)
}
