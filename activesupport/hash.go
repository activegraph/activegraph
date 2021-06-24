package activesupport

type Hash map[string]interface{}

type HashConverter interface {
	ToHash() Hash
}

type HashArrayConverter interface {
	ToHashArray() []Hash
}
