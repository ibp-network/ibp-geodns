package api

import (
	"net/http"

	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"
	types "ibp-geodns/src/mgmtApi/api/types"
)

// ApiQuery_Member handles /api/member requests
func ApiQuery_Member(w http.ResponseWriter, r *http.Request, req types.Request) types.Response {
	switch req.Method {
	case "list":
		return apiQuery_MemberList(req)
	default:
		return types.Response{Error: "Invalid member request"}
	}
}

func apiQuery_MemberList(params types.Request) types.Response {
	c := cfg.GetConfig()
	var memberList []cfg.Member
	for _, member := range c.Members {
		memberList = append(memberList, member)
	}
	return types.Response{Result: memberList}
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
