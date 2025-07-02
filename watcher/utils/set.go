package utils

// StringSliceToMap converts a slice of strings into a map with keys from the slice
func StringSliceToMap(strings []string) map[string]bool {
	ret := map[string]bool{}
	for _, s := range strings {
		ret[s] = true
	}
	return ret
}
