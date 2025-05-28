package monitor

// getIntOption returns an integer from ExtraOptions or the default
func getIntOption(extraOptions map[string]interface{}, key string, defaultValue int) int {
	if val, ok := extraOptions[key].(float64); ok {
		return int(val)
	}
	return defaultValue
}

// getFloatOption returns a float64 from ExtraOptions or the default
func getFloatOption(extraOptions map[string]interface{}, key string, defaultValue float64) float64 {
	if val, ok := extraOptions[key].(float64); ok {
		return val
	}
	return defaultValue
}
