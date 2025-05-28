package nats

import (
	cfg "ibp-geodns/src/common/config"
)

// findMemberByName searches for a member by name.
func findMemberByName(memberName string) (cfg.Member, bool) {
	c := cfg.GetConfig()
	for _, m := range c.Members {
		if m.Details.Name == memberName {
			return m, true
		}
	}
	return cfg.Member{}, false
}
