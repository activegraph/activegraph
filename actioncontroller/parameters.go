package actioncontroller

type Parameters map[string]interface{}

func (p Parameters) Get(key string) Parameters {
	val, ok := p[key]
	if !ok {
		return nil
	}
	params, ok := val.(map[string]interface{})
	if !ok {
		return nil
	}
	return params
}

func (p Parameters) ToH() map[string]interface{} {
	return (map[string]interface{})(p)
}
