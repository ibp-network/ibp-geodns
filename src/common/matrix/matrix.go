package matrix

import (
	"context"
	"fmt"
	"sync"
	"time"

	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"

	mautrix "maunium.net/go/mautrix"
	"maunium.net/go/mautrix/id"
)

var (
	client     *mautrix.Client
	clientOnce sync.Once
	roomID     id.RoomID
)

func Init() {
	clientOnce.Do(initInternal)
}

func initInternal() {
	c := cfg.GetConfig()

	mxCfg := c.Local.Matrix
	if mxCfg.HomeServerURL == "" || mxCfg.Username == "" || mxCfg.Password == "" || mxCfg.RoomID == "" {
		log.Log(log.Warn, "[matrix] configuration incomplete – Matrix notifications disabled")
		return
	}

	var err error
	client, err = mautrix.NewClient(mxCfg.HomeServerURL, "", "")
	if err != nil {
		log.Log(log.Error, "[matrix] failed to create client: %v", err)
		client = nil
		return
	}

	ctx := context.Background()
	_, err = client.Login(ctx, &mautrix.ReqLogin{
		Type: "m.login.password",
		Identifier: mautrix.UserIdentifier{
			Type: mautrix.IdentifierTypeUser,
			User: mxCfg.Username,
		},
		Password: mxCfg.Password,
	})
	if err != nil {
		log.Log(log.Error, "[matrix] login failed: %v", err)
		client = nil
		return
	}

	roomID = id.RoomID(mxCfg.RoomID)
	log.Log(log.Info, "[matrix] logged in as %s, ready to send alerts to %s", mxCfg.Username, roomID)
}

func NotifyMemberOffline(memberName, checkType, checkName, domainName, endpoint string, isIPv6 bool, errText string, when time.Time) {
	if client == nil {
		return
	}

	msg := fmt.Sprintf(
		"🚨 **Member offline**\n"+
			"- **Name:** %s\n"+
			"- **Check:** %s / %s\n"+
			"- **Domain:** %s\n"+
			"- **Endpoint:** %s\n"+
			"- **IPv6:** %v\n"+
			"- **Time (UTC):** %s\n"+
			"- **Error:** %s",
		memberName, checkType, checkName, domainName, endpoint, isIPv6,
		when.UTC().Format(time.RFC3339), errText)

	ctx := context.Background()
	_, err := client.SendText(ctx, roomID, msg)
	if err != nil {
		log.Log(log.Error, "[matrix] failed to send alert: %v", err)
	}
}
