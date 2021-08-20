package activesupport

type Initializer interface {
	Initialize() error
}
