package nats

func Disconnect() {
	natsMu.Lock()
	defer natsMu.Unlock()
	if nc != nil && !nc.IsClosed() {
		nc.Close()
		nc = nil
	}
}
