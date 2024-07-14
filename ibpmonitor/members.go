package ibpmonitor

func (r *IbpMonitor) AddMember(newMember Member) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Members = append(r.Members, newMember)
}

func (r *IbpMonitor) RemoveMember(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, member := range r.Members {
		if member.MemberName == name {
			r.Members = append(r.Members[:i], r.Members[i+1:]...)
			break
		}
	}
}
