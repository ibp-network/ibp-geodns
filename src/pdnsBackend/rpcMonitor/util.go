package rpcMonitor

func getIntOption(extraOptions map[string]interface{}, key string, defaultValue int) int {
	if value, ok := extraOptions[key].(float64); ok {
		return int(value)
	}
	return defaultValue
}

func getFloatOption(extraOptions map[string]interface{}, key string, defaultValue float64) float64 {
	if value, ok := extraOptions[key].(float64); ok {
		return value
	}
	return defaultValue
}
