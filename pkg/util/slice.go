package util

// StringSliceContains checks if  string slice contains given string
func StringSliceContains(slice []string, str string) bool {
	for _, a := range slice {
		if a == str {
			return true
		}
	}
	return false
}
