package monitor

func getIntOption(extraOptions map[string]interface{}, key string, defaultValue int) int {
	if val, ok := extraOptions[key].(float64); ok {
		return int(val)
	}
	return defaultValue
}

func getFloatOption(extraOptions map[string]interface{}, key string, defaultValue float64) float64 {
	if val, ok := extraOptions[key].(float64); ok {
		return val
	}
	return defaultValue
}
