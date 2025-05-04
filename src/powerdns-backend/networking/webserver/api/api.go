package api

import (
	"common/config"
	l "common/logging"
)

func Init() {

}

// checkMemberAuth ensures that the provided member name and auth key match
// the configuration in c.System.APIServer.AuthKeys.
func checkMemberAuth(c config.ConfigData, memberName, authKey string) bool {
	expectedKey, exists := c.System.MgmtApi.AuthKeys[memberName]
	if !exists {
		l.Log(l.Warn, "Auth failed: memberName %s not found in AuthKeys", memberName)
		return false
	}

	if expectedKey != authKey {
		l.Log(l.Warn, "Auth failed: provided authKey does not match for member %s", memberName)
		return false
	}

	return true
}
