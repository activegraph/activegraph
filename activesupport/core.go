package activesupport

type Initializer interface {
	Initialize() error
}

func IsBlank(value interface{}) bool {
	var s String

	switch value := value.(type) {
	case bool:
		return value
	case string:
		s = String(value)
	case []rune:
		s = String(string(value))
	case []byte:
		s = String(string(value))
	default:
		return false
	}
	return s.IsBlank()
}
