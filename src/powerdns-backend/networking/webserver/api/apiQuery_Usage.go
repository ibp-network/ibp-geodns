package api

import (
	"encoding/json"
	"ibp-geodns/src/common/data"
	l "ibp-geodns/src/common/logging"
	"net/http"
)

func ApiQuery_Usage(w http.ResponseWriter, r *http.Request) ApiResponse {
	var req ApiRequest
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&req)
	if err != nil {
		l.Log(l.Error, "Bad request: %v", http.StatusBadRequest)
		return ApiResponse{Result: ""}
	}

	var res ApiResponse

	switch req.Method {
	case "byClassc":
		res = apiQuery_UsageByClassc()
	case "byCountry":
		res = apiQuery_UsageByCountry()
	case "byMember":
		res = apiQuery_UsageByMember()
	case "byDomain":
		res = apiQuery_UsageByDomain()
	default:
		res = ApiResponse{Error: "Invalid request"}
	}

	return res
}

func apiQuery_UsageByCountry() ApiResponse {
	s := data.GetStats()

	// Structure: date -> country -> { members, classcs, domains }
	result := make(map[string]map[string]struct {
		Members map[string]int `json:"members"`
		ClassCs map[string]int `json:"classcs"`
		Domains map[string]int `json:"domains"`
	})

	for date, domains := range s {
		if _, ok := result[date]; !ok {
			result[date] = make(map[string]struct {
				Members map[string]int `json:"members"`
				ClassCs map[string]int `json:"classcs"`
				Domains map[string]int `json:"domains"`
			})
		}

		for domainName, dailyStats := range domains {
			// ClientStats first
			for ctry, cCount := range dailyStats.ClientStats.Countries {
				ctryStruct, ok := result[date][ctry]
				if !ok {
					ctryStruct = struct {
						Members map[string]int `json:"members"`
						ClassCs map[string]int `json:"classcs"`
						Domains map[string]int `json:"domains"`
					}{
						Members: map[string]int{},
						ClassCs: map[string]int{},
						Domains: map[string]int{},
					}
				}
				// Domain hits from client perspective
				ctryStruct.Domains[domainName] += cCount

				// ClassC from clientStats
				for classc, ccCount := range dailyStats.ClientStats.ClassCs {
					ctryStruct.ClassCs[classc] += ccCount
				}

				result[date][ctry] = ctryStruct
			}

			// MemberStats now
			for memberName, ms := range dailyStats.MemberStats {
				for ctry, cCount := range ms.Countries {
					ctryStruct, ok := result[date][ctry]
					if !ok {
						ctryStruct = struct {
							Members map[string]int `json:"members"`
							ClassCs map[string]int `json:"classcs"`
							Domains map[string]int `json:"domains"`
						}{
							Members: map[string]int{},
							ClassCs: map[string]int{},
							Domains: map[string]int{},
						}
					}
					// Members hits
					ctryStruct.Members[memberName] += cCount

					// ClassCs from memberStats
					for classc, ccCount := range ms.ClassCs {
						ctryStruct.ClassCs[classc] += ccCount
					}

					// Domains from member perspective (this might not be distinct from client domains,
					// but if you want to sum them, we add here as well)
					ctryStruct.Domains[domainName] += ms.Requests

					result[date][ctry] = ctryStruct
				}
			}
		}
	}

	return ApiResponse{Result: result}
}

func apiQuery_UsageByClassc() ApiResponse {
	s := data.GetStats()

	// Structure: date -> classc -> { members, countries, domains }
	result := make(map[string]map[string]struct {
		Members   map[string]int `json:"members"`
		Countries map[string]int `json:"countries"`
		Domains   map[string]int `json:"domains"`
	})

	for date, domains := range s {
		if _, ok := result[date]; !ok {
			result[date] = make(map[string]struct {
				Members   map[string]int `json:"members"`
				Countries map[string]int `json:"countries"`
				Domains   map[string]int `json:"domains"`
			})
		}

		for domainName, dailyStats := range domains {
			// From ClientStats
			for classc, ccCount := range dailyStats.ClientStats.ClassCs {
				classcStruct, ok := result[date][classc]
				if !ok {
					classcStruct = struct {
						Members   map[string]int `json:"members"`
						Countries map[string]int `json:"countries"`
						Domains   map[string]int `json:"domains"`
					}{
						Members:   map[string]int{},
						Countries: map[string]int{},
						Domains:   map[string]int{},
					}
				}

				// Add domains from client perspective
				classcStruct.Domains[domainName] += ccCount

				// Add countries from client perspective
				for ctry, cCount := range dailyStats.ClientStats.Countries {
					classcStruct.Countries[ctry] += cCount
				}

				result[date][classc] = classcStruct
			}

			// From MemberStats
			for memberName, ms := range dailyStats.MemberStats {
				for classc, ccCount := range ms.ClassCs {
					classcStruct, ok := result[date][classc]
					if !ok {
						classcStruct = struct {
							Members   map[string]int `json:"members"`
							Countries map[string]int `json:"countries"`
							Domains   map[string]int `json:"domains"`
						}{
							Members:   map[string]int{},
							Countries: map[string]int{},
							Domains:   map[string]int{},
						}
					}

					// Members
					classcStruct.Members[memberName] += ccCount

					// Countries
					for ctry, cCount := range ms.Countries {
						classcStruct.Countries[ctry] += cCount
					}

					// Domains
					classcStruct.Domains[domainName] += ms.Requests

					result[date][classc] = classcStruct
				}
			}
		}
	}

	return ApiResponse{Result: result}
}

func apiQuery_UsageByMember() ApiResponse {
	s := data.GetStats()

	// Structure: date -> memberName -> { classcs, countries, domains }
	result := make(map[string]map[string]struct {
		ClassCs   map[string]int `json:"classcs"`
		Countries map[string]int `json:"countries"`
		Domains   map[string]int `json:"domains"`
	})

	for date, domains := range s {
		if _, ok := result[date]; !ok {
			result[date] = make(map[string]struct {
				ClassCs   map[string]int `json:"classcs"`
				Countries map[string]int `json:"countries"`
				Domains   map[string]int `json:"domains"`
			})
		}

		for domainName, dailyStats := range domains {
			// We can also consider client stats as non-member hits. If needed, ignore or handle separately.
			// For usage by member, we focus primarily on MemberStats.
			for memberName, ms := range dailyStats.MemberStats {
				memberStruct, ok := result[date][memberName]
				if !ok {
					memberStruct = struct {
						ClassCs   map[string]int `json:"classcs"`
						Countries map[string]int `json:"countries"`
						Domains   map[string]int `json:"domains"`
					}{
						ClassCs:   map[string]int{},
						Countries: map[string]int{},
						Domains:   map[string]int{},
					}
				}

				// Aggregate ClassCs
				for classc, ccCount := range ms.ClassCs {
					memberStruct.ClassCs[classc] += ccCount
				}

				// Aggregate Countries
				for ctry, cCount := range ms.Countries {
					memberStruct.Countries[ctry] += cCount
				}

				// Aggregate Domains
				memberStruct.Domains[domainName] += ms.Requests

				result[date][memberName] = memberStruct
			}
		}
	}

	return ApiResponse{Result: result}
}

func apiQuery_UsageByDomain() ApiResponse {
	s := data.GetStats()

	// Structure: date -> domain -> { members, classcs, countries }
	result := make(map[string]map[string]struct {
		Members   map[string]int `json:"members"`
		ClassCs   map[string]int `json:"classcs"`
		Countries map[string]int `json:"countries"`
	})

	for date, domains := range s {
		if _, ok := result[date]; !ok {
			result[date] = make(map[string]struct {
				Members   map[string]int `json:"members"`
				ClassCs   map[string]int `json:"classcs"`
				Countries map[string]int `json:"countries"`
			})
		}

		for domainName, dailyStats := range domains {
			domainStruct, ok := result[date][domainName]
			if !ok {
				domainStruct = struct {
					Members   map[string]int `json:"members"`
					ClassCs   map[string]int `json:"classcs"`
					Countries map[string]int `json:"countries"`
				}{
					Members:   map[string]int{},
					ClassCs:   map[string]int{},
					Countries: map[string]int{},
				}
			}

			// From clientStats
			for classc, ccCount := range dailyStats.ClientStats.ClassCs {
				domainStruct.ClassCs[classc] += ccCount
			}
			for ctry, cCount := range dailyStats.ClientStats.Countries {
				domainStruct.Countries[ctry] += cCount
			}

			// From memberStats
			for memberName, ms := range dailyStats.MemberStats {
				domainStruct.Members[memberName] += ms.Requests

				// Add member classcs and countries as well to the domain's totals
				for classc, ccCount := range ms.ClassCs {
					domainStruct.ClassCs[classc] += ccCount
				}
				for ctry, cCount := range ms.Countries {
					domainStruct.Countries[ctry] += cCount
				}
			}

			result[date][domainName] = domainStruct
		}
	}

	return ApiResponse{Result: result}
}
