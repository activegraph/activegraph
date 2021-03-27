package internal

func CopyMap(m map[string]interface{}) map[string]interface{} {
	mout := make(map[string]interface{})
	for k, v := range m {
		mout[k] = v
	}
	return mout
}
