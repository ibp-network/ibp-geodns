package api

import (
	"net/http"

	cfg "ibp-geodns/src/common/config"
	dat "ibp-geodns/src/common/data"
	log "ibp-geodns/src/common/logging"
)

// ApiQuery_Member handles /api/member requests
func ApiQuery_Member(w http.ResponseWriter, r *http.Request, req ApiRequest) ApiResponse {
	switch req.Method {
	case "enable":
		return apiQuery_MemberEnable(req)
	case "disable":
		return apiQuery_MemberDisable(req)
	case "list":
		return apiQuery_MemberList(req)
	default:
		return ApiResponse{Error: "Invalid member request"}
	}
}

func apiQuery_MemberEnable(params ApiRequest) ApiResponse {
	c := cfg.GetConfig()
	if !checkMemberAuth(c, params.MemberName, params.AuthKey) {
		return ApiResponse{Error: "Authentication failed"}
	}
	dat.MemberEnable(params.MemberName)
	return ApiResponse{Result: "Member enabled successfully"}
}

func apiQuery_MemberDisable(params ApiRequest) ApiResponse {
	c := cfg.GetConfig()
	if !checkMemberAuth(c, params.MemberName, params.AuthKey) {
		return ApiResponse{Error: "Authentication failed"}
	}
	dat.MemberDisable(params.MemberName)
	return ApiResponse{Result: "Member disabled successfully"}
}

func apiQuery_MemberList(params ApiRequest) ApiResponse {
	c := cfg.GetConfig()
	var memberList []cfg.Member
	for _, member := range c.Members {
		memberList = append(memberList, member)
	}
	return ApiResponse{Result: memberList}
}

// checkMemberAuth ensures the (MemberName, AuthKey) pair is valid
func checkMemberAuth(c cfg.Config, memberName, authKey string) bool {
	expectedKey, exists := c.Local.MgmtApi.AuthKeys[memberName]
	if !exists {
		log.Log(log.Warn, "Auth failed: no AuthKey found for memberName %s", memberName)
		return false
	}
	if expectedKey != authKey {
		log.Log(log.Warn, "Auth failed: AuthKey mismatch for member %s", memberName)
		return false
	}
	return true
}
