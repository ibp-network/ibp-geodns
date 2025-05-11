package nats

import (
	cfg "ibp-geodns/src/common/config"
)

// findCheckByName searches for a check by name and type.
func findCheckByName(checkName, checkType string) (cfg.Check, bool) {
	c := cfg.GetConfig()
	for _, ch := range c.Local.Checks {
		if ch.Name == checkName && ch.CheckType == checkType {
			return ch, true
		}
	}
	return cfg.Check{}, false
}
