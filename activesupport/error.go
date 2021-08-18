package activesupport

import (
	"fmt"
)

type ErrMultipleVariadicArguments struct {
	Name string
}

func (e *ErrMultipleVariadicArguments) Error() string {
	return fmt.Sprintf("Only one variadic argument for '%s' permitted", e.Name)
}

// ErrArgument is returned when the arguments are wrong.
type ErrArgument struct {
	Message string
}

// Error implements error interafce and return human-readable error message.
func (e ErrArgument) Error() string {
	return fmt.Sprintf("ErrArgument: %s", e.Message)
}
