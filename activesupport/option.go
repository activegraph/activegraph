package activesupport

import (
	"fmt"
)

type Option[T any] struct {
	some *T
	none bool
}

func None[T any]() Option[T] {
	return Option[T]{new(T), true}
}

func Some[T any](val T) Option[T] {
	return Option[T]{&val, false}
}

func (o Option[T]) String() string {
	if o.none {
		return "None"
	}
	return fmt.Sprintf("Some(%v)", *o.some)
}

func (o Option[T]) IsNone() bool {
	return o.none
}

func (o Option[T]) IsSome() bool {
	return !o.none
}

func (o Option[T]) Unwrap() T {
	if o.none {
		panic("called `Option.Unwrap` on a `None` value")
	}
	return *o.some
}

func (o Option[T]) UnwrapOr(val T) T {
	if o.none {
		return val
	}
	return *o.some
}
