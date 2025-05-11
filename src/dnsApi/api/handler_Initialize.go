package api

import (
	"net/http"
)

// Handle initialization call (not necessary for us to do anything)
func handle_Init(w http.ResponseWriter, r *http.Request, req Request) Response {
	return Response{Result: true}
}
