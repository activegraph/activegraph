package activerecord

type Ownership interface {
	Move(dst interface{}) error
	Borrow(src interface{}) error
}
