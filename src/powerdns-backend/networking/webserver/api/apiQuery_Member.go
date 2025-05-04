package api

import (
	"encoding/json"
	"ibp-geodns/config"
	"ibp-geodns/data"
	l "ibp-geodns/logging"
	"net/http"
)

func ApiQuery_Member(w http.ResponseWriter, r *http.Request) ApiResponse {
	var req ApiRequest
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&req)
	if err != nil {
		l.Log(l.Error, "Bad request: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return ApiResponse{Result: ""}
	}

	var res ApiResponse

	switch req.Method {
	case "enable":
		res = apiQuery_MemberEnable(req)
	case "disable":
		res = apiQuery_MemberDisable(req)
	case "list":
		res = apiQuery_MemberList()
	default:
		res = ApiResponse{Error: "Invalid request"}
	}

	return res
}

func apiQuery_MemberEnable(params ApiRequest) ApiResponse {
	c := config.GetConfig()

	// Check authentication
	if !checkMemberAuth(c, params.MemberName, params.AuthKey) {
		return ApiResponse{Error: "Authentication failed"}
	}

	// Enable the member
	data.MemberEnable(params.MemberName)

	return ApiResponse{Result: "Member enabled successfully"}
}

func apiQuery_MemberDisable(params ApiRequest) ApiResponse {
	c := config.GetConfig()

	// Check authentication
	if !checkMemberAuth(c, params.MemberName, params.AuthKey) {
		return ApiResponse{Error: "Authentication failed"}
	}

	// Disable the member
	data.MemberDisable(params.MemberName)

	return ApiResponse{Result: "Member disabled successfully"}
}

func apiQuery_MemberList() ApiResponse {
	c := config.GetConfig()

	var memberList []config.Member
	for _, member := range c.Members {
		memberList = append(memberList, member)
	}

	return ApiResponse{Result: memberList}
}
