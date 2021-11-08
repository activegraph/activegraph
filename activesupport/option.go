package activesupport

import (
	"fmt"
)

type T interface {
	Default() interface{}
}

type Option struct {
	t T

	some interface{}
	none bool
}

func None(t T) Option {
	return Option{t, nil, true}
}

func Some(t T, val interface{}) Option {
	if t == nil {
		panic("some is nil")
	}
	return Option{t, val, false}
}

func (o Option) String() string {
	if o.none {
		return "None"
	}
	return fmt.Sprintf("Some(%s)", o.some)
}

func (o Option) IsNone() bool {
	return o.none
}

func (o Option) Unwrap() interface{} {
	if o.none {
		panic("called `Option.Unwrap` on a `None` value")
	}
	return o.some
}

func (o Option) UnwrapOr(val interface{}) interface{} {
	if o.none {
		return val
	}
	return o.some
}

func (o Option) UnwrapOrDefault() interface{} {
	if o.none {
		return o.t.Default()
	}
	return o.some
}
