package mgmtApi

import (
	"net/http"

	"ibp-geodns/src/common/config"
	"ibp-geodns/src/common/data"
	l "ibp-geodns/src/common/logging"
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
	c := config.GetConfig()
	if !checkMemberAuth(c, params.MemberName, params.AuthKey) {
		return ApiResponse{Error: "Authentication failed"}
	}
	data.MemberEnable(params.MemberName)
	return ApiResponse{Result: "Member enabled successfully"}
}

func apiQuery_MemberDisable(params ApiRequest) ApiResponse {
	c := config.GetConfig()
	if !checkMemberAuth(c, params.MemberName, params.AuthKey) {
		return ApiResponse{Error: "Authentication failed"}
	}
	data.MemberDisable(params.MemberName)
	return ApiResponse{Result: "Member disabled successfully"}
}

func apiQuery_MemberList(params ApiRequest) ApiResponse {
	c := config.GetConfig()
	var memberList []config.Member
	for _, member := range c.Members {
		memberList = append(memberList, member)
	}
	return ApiResponse{Result: memberList}
}

// checkMemberAuth ensures the (MemberName, AuthKey) pair is valid
func checkMemberAuth(c config.ConfigData, memberName, authKey string) bool {
	expectedKey, exists := c.System.MgmtApi.AuthKeys[memberName]
	if !exists {
		l.Log(l.Warn, "Auth failed: no AuthKey found for memberName %s", memberName)
		return false
	}
	if expectedKey != authKey {
		l.Log(l.Warn, "Auth failed: AuthKey mismatch for member %s", memberName)
		return false
	}
	return true
}
