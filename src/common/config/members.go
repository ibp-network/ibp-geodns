package config

func GetMember(name string) (Member, bool) {
	cfg.mu.RLock()
	member, exists := cfg.data.Members[name]
	cfg.mu.RUnlock()
	return member, exists
}

func SetMember(name string, member Member) {
	cfg.mu.Lock()
	cfg.data.Members[name] = member
	cfg.mu.Unlock()
}

func DeleteMember(name string) {
	cfg.mu.Lock()
	delete(cfg.data.Members, name)
	cfg.mu.Unlock()
}

func ListMembers() map[string]Member {
	cfg.mu.RLock()
	copy := make(map[string]Member, len(cfg.data.Members))
	for k, v := range cfg.data.Members {
		copy[k] = v
	}
	cfg.mu.RUnlock()
	return copy
}
