package activesupport

type HashConverter interface {
	ToHash() map[string]interface{}
}
