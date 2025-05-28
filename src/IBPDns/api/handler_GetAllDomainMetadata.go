package api

func handle_GetAllDomainMetadata(req Request) Response {
	// PDNS usually expects a slice of domain metadata objects. Returning an empty slice
	// will satisfy its request without causing errors.
	// Example structure: each object might have fields { id, kind, content }, etc.
	// But for now, we just return an empty list.

	// If you needed actual metadata, you'd gather it here.
	return Response{Result: []interface{}{}}
}
