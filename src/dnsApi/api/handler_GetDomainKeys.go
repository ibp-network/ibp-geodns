package api

import (
	"net/http"
)

// dnsQuery_GetDomainKeys retrieves DNSKEY records for a given domain.
func handle_GetDomainKeys(w http.ResponseWriter, r *http.Request, req Request) Response {
	for _, domain := range TLDRecords.records {
		if req.Parameters.QName == domain {
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
	}

	return Response{Result: nil}
}
