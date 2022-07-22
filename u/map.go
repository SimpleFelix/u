package u

// MapGet 返回key对应的值。如果找不到key，或者值类型不匹配，则返回defaultValue。
// todo generic
func MapGet(m map[string]interface{}, key string, defaultValue interface{}) interface{} {
	v, ok := m[key]
	if !ok {
		return defaultValue
	}
	return v
	//t := reflect.TypeOf(defaultValue)
	//vv, ok := v.(t)
	//if !ok {
	//	return defaultValue
	//}
	//return vv
}

// MapGetString 返回key对应的值。如果找不到key，或者值类型不匹配，则返回defaultValue。
func MapGetString(m map[string]interface{}, key string, defaultValue string) string {
	v, ok := m[key]
	if !ok {
		return defaultValue
	}

	tv, ok := v.(string)
	if !ok {
		return defaultValue
	}
	return tv
}

// MapGetInt 返回key对应的值。如果找不到key，或者值类型不匹配，则返回defaultValue。
func MapGetInt(m map[string]interface{}, key string, defaultValue int) int {
	v, ok := m[key]
	if !ok {
		return defaultValue
	}

	tv, ok := v.(int)
	if !ok {
		return defaultValue
	}
	return tv
}

// MapGetFloat64 返回key对应的值。如果找不到key，或者值类型不匹配，则返回defaultValue。
func MapGetFloat64(m map[string]interface{}, key string, defaultValue float64) float64 {
	v, ok := m[key]
	if !ok {
		return defaultValue
	}

	tv, ok := v.(float64)
	if !ok {
		return defaultValue
	}
	return tv
}
