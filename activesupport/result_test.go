package activesupport

import (
	"testing"
)

func TestFutureResult_AndThen(t *testing.T) {
	var res Result[int] = FutureOk(10)

	res = res.AndThen(func(val int) Result[int] {
		return Ok(15)
	})

	res.Expect("expression should return 15")
	if !res.Contains(15) {
		t.Fatalf("%v != %v", res.Ok(), Some(15))
	}
}
