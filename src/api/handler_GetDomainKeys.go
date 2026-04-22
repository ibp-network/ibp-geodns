package api

func handle_GetDomainKeys(req Request) Response {
	name := req.Parameters.Name
	if name == "" {
		name = req.Parameters.QName
	}

	_, _, ok := findZone(name)
	if !ok {
		return Response{Result: nil}
	}

	// Do not advertise placeholder DNSSEC keys. Until real signing material exists,
	// return an empty keyset so PowerDNS does not publish bogus DNSKEY records.
	return Response{Result: []struct{}{}}
}
