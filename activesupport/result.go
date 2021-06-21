package activesupport

type Res interface {
	Ok() interface{}
	Err() error
}

type res struct {
	ok  interface{}
	err error
}

func (r res) Ok() interface{} {
	if r.err != nil {
		return nil
	}
	return r.ok
}

func (r res) Err() error {
	if r.err != nil {
		return r.err
	}
	return nil
}

func Result(val interface{}, err error) Res {
	return res{ok: val, err: err}
}
