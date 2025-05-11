package handlers

import "ibp-geodns/src/mgmtApi/api/types"

var (
	ServiceRecords types.ServiceMap
	StaticRecords  types.StaticMap
	TLDRecords     types.TLDMap
)

func init(service types.ServiceMap, static types.StaticMap, tld types.TLDMap) {

	ServiceRecords = service
	StaticRecords = static
	TLDRecords = tld

}
