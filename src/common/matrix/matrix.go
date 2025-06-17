package matrix

import (
	"context"
	"fmt"
	"sync"
	"time"

	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"

	mautrix "maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

var (
	client     *mautrix.Client
	roomID     id.RoomID
	clientOnce sync.Once
)

type offlineKey struct {
	Member, CheckType, CheckName, Domain, Endpoint string
	IsIPv6                                         bool
}

var offlineMsgCache sync.Map

func Init() { clientOnce.Do(initInternal) }

func initInternal() {
	c := cfg.GetConfig()
	mx := c.Local.Matrix
	if mx.HomeServerURL == "" || mx.Username == "" || mx.Password == "" || mx.RoomID == "" {
		log.Log(log.Warn, "[matrix] configuration incomplete – chat notifications disabled")
		return
	}

	var err error
	client, err = mautrix.NewClient(mx.HomeServerURL, "", "")
	if err != nil {
		log.Log(log.Error, "[matrix] cannot create client: %v", err)
		client = nil
		return
	}

	ctx := context.Background()
	_, err = client.Login(ctx, &mautrix.ReqLogin{
		Type: "m.login.password",
		Identifier: mautrix.UserIdentifier{
			Type: mautrix.IdentifierTypeUser,
			User: mx.Username,
		},
		Password: mx.Password,
	})
	if err != nil {
		log.Log(log.Error, "[matrix] login failed: %v", err)
		client = nil
		return
	}

	roomID = id.RoomID(mx.RoomID)
	log.Log(log.Info, "[matrix] logged in as %s – alerts will be sent to %s", mx.Username, roomID)
}

func NotifyMemberOffline(member, checkType, checkName, domain, endpoint string, isIPv6 bool, errText string, when time.Time) {
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
		member, checkType, checkName, domain, endpoint, isIPv6,
		when.UTC().Format(time.RFC3339), errText)

	ctx := context.Background()
	resp, err := client.SendText(ctx, roomID, msg)
	if err != nil {
		log.Log(log.Error, "[matrix] failed to send offline alert: %v", err)
		return
	}

	key := offlineKey{
		Member:    member,
		CheckType: checkType, CheckName: checkName,
		Domain: domain, Endpoint: endpoint, IsIPv6: isIPv6,
	}
	offlineMsgCache.Store(key, resp.EventID)
}

func NotifyMemberOnline(member, checkType, checkName, domain, endpoint string, isIPv6 bool, when time.Time) {
	if client == nil {
		return
	}
	key := offlineKey{
		Member:    member,
		CheckType: checkType, CheckName: checkName,
		Domain: domain, Endpoint: endpoint, IsIPv6: isIPv6,
	}

	evt, ok := offlineMsgCache.Load(key)
	if !ok {
		msg := fmt.Sprintf("✅ %s is back online (%s/%s, %s, IPv6=%v) – %s",
			member, checkType, checkName, domain, isIPv6,
			when.UTC().Format(time.RFC3339))
		ctx := context.Background()
		_, _ = client.SendText(ctx, roomID, msg)
		return
	}

	eventID := evt.(id.EventID)
	newBody := fmt.Sprintf(
		"✅ **Member back online**\n"+
			"- **Name:** %s\n"+
			"- **Check:** %s / %s\n"+
			"- **Domain:** %s\n"+
			"- **Endpoint:** %s\n"+
			"- **IPv6:** %v\n"+
			"- **Recovered (UTC):** %s",
		member, checkType, checkName, domain, endpoint, isIPv6,
		when.UTC().Format(time.RFC3339))

	ctx := context.Background()
	if err := sendReplacement(ctx, roomID, eventID, newBody); err != nil {
		log.Log(log.Error, "[matrix] edit failed (%v), sending new message", err)
		_, _ = client.SendText(ctx, roomID, newBody)
	}
	offlineMsgCache.Delete(key)
}

func sendReplacement(ctx context.Context, room id.RoomID, target id.EventID, newBody string) error {
	content := map[string]interface{}{
		"msgtype": "m.text",
		"body":    "* " + newBody,
		"m.new_content": map[string]interface{}{
			"msgtype": "m.text",
			"body":    newBody,
		},
		"m.relates_to": map[string]interface{}{
			"rel_type": "m.replace",
			"event_id": target,
		},
	}

	_, err := client.SendMessageEvent(ctx, room, event.EventMessage, content)
	return err
}
