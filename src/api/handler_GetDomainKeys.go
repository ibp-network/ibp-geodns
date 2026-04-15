package api

func handle_GetDomainKeys(req Request) Response {
	name := req.Parameters.Name
	if name == "" {
		name = req.Parameters.QName
	}

	_, domain, ok := findZone(name)
	if !ok {
		return Response{Result: nil}
	}

	keys := []struct {
		ID        int    `json:"id"`
		Flags     int    `json:"flags"`
		Active    bool   `json:"active"`
		Published bool   `json:"published"`
		Content   string `json:"content"`
	}{{
		ID:        3,
		Flags:     257,
		Active:    true,
		Published: true,
		Content:   domain + " IN DNSKEY 257 3 13 Ts7EglQbnyZDVklFGoiAnbB/DGzlJC4RBft7/wouiSxgQ9OB7sXD9yOkhyjhs5BzaOFs0LivpUwQZnYFkafAYA==",
	}}

	return Response{Result: keys}
}
