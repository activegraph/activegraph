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
