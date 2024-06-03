package maputil

import "fmt"

// ConvertInterfaceMap converts map[interface{}]interface{} to
// map[string]interface{} with use with an Unmarshaler.  This is required
// because yaml.v2 unmarshals maps to map[interface{}]interface{}.
func ConvertInterfaceMap(in map[interface{}]interface{}) map[string]interface{} {
	res := make(map[string]interface{})
	for k, v := range in {
		res[fmt.Sprintf("%v", k)] = cleanupMapValue(v)
	}
	return res
}

func cleanupMapValue(v interface{}) interface{} {
	switch v := v.(type) {
	case map[interface{}]interface{}:
		return ConvertInterfaceMap(v)
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}
