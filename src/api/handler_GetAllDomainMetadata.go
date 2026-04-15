package api

func handle_GetAllDomainMetadata(req Request) Response {
	name := req.Parameters.Name
	if name == "" {
		name = req.Parameters.QName
	}

	return Response{Result: domainMetadataForZone(name)}
}
