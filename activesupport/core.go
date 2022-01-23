package activesupport

type Initializer interface {
	Initialize() error
}

func IsBlank(value interface{}) bool {
	var s Str

	switch value := value.(type) {
	case bool:
		return value
	case string:
		s = Str(value)
	case []rune:
		s = Str(string(value))
	case []byte:
		s = Str(string(value))
	case nil:
		return true
	default:
		return false
	}
	return s.IsBlank()
}
