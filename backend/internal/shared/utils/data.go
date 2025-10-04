package utils

// ConvertStringMapToInterfaceMap converts map[string]string to map[string]any
func ConvertStringMapToInterfaceMap(m map[string]string) map[string]any {
	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}
