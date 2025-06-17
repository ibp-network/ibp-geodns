package matrix

import (
	"context"
	"fmt"
	"sync"
	"time"

	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

// -----------------------------------------------------------------------------
// PACKAGE‑LEVEL STATE
// -----------------------------------------------------------------------------
var (
	client     *mautrix.Client // logged‑in Matrix client
	userID     id.UserID       // the local user (after login)
	roomID     id.RoomID       // destination room
	once       sync.Once       // protect Init()
	offlineMap sync.Map        // key → id.EventID (for edit‑in‑place)
)

// -----------------------------------------------------------------------------
// INITIALISATION
// -----------------------------------------------------------------------------
func Init() {
	once.Do(func() {
		c := cfg.GetConfig().Local.Matrix
		if c.HomeServerURL == "" || c.Username == "" ||
			c.Password == "" || c.RoomID == "" {
			log.Log(log.Warn, "[matrix] configuration incomplete – Matrix integration disabled")
			return
		}

		cli, err := mautrix.NewClient(c.HomeServerURL, "", "")
		if err != nil {
			log.Log(log.Error, "[matrix] create client: %v", err)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		resp, err := cli.Login(ctx, &mautrix.ReqLogin{
			Type: "m.login.password",
			Identifier: mautrix.UserIdentifier{
				Type: "m.id.user",
				User: c.Username,
			},
			Password: c.Password,
		})
		if err != nil {
			log.Log(log.Error, "[matrix] login failed: %v", err)
			return
		}

		cli.SetCredentials(resp.UserID, resp.AccessToken)
		client = cli
		userID = resp.UserID
		roomID = id.RoomID(c.RoomID)

		log.Log(log.Info, "[matrix] logged in as %s; ready to post to %s", userID, roomID)
	})
}

// isReady verifies we have a usable, authenticated client.
func isReady() bool {
	return client != nil && client.AccessToken != ""
}

// -----------------------------------------------------------------------------
// INTERNAL HELPERS
// -----------------------------------------------------------------------------
func makeKey(member, checkType, checkName, domain, endpoint string, ipv6 bool) string {
	return fmt.Sprintf("%s|%s|%s|%s|%s|%v",
		member, checkType, checkName, domain, endpoint, ipv6)
}

// sendText posts a plain‑text message and returns the event ID.
func sendText(ctx context.Context, body string) (id.EventID, error) {
	resp, err := client.SendText(ctx, roomID, body)
	if err != nil {
		return "", err
	}
	return resp.EventID, nil
}

// editText attempts to *replace* (MSC‑2674) an existing event.
func editText(ctx context.Context, target id.EventID, body string) error {
	content := map[string]interface{}{
		"msgtype": "m.text",
		"body":    body,
		"m.new_content": map[string]interface{}{
			"msgtype": "m.text",
			"body":    body,
		},
		"m.relates_to": map[string]interface{}{
			"rel_type": "m.replace",
			"event_id": target,
		},
	}

	_, err := client.SendMessageEvent(ctx, roomID, event.EventMessage, content)
	return err
}

// -----------------------------------------------------------------------------
// PUBLIC NOTIFICATION API
// -----------------------------------------------------------------------------
func NotifyMemberOffline(
	member, checkType, checkName, domain, endpoint string,
	ipv6 bool, errText string,
) {
	if !isReady() {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	body := fmt.Sprintf(
		"⚠️  *OFFLINE*\n"+
			"• Member: **%s**\n"+
			"• Check:  %s / %s\n"+
			"• Domain: %s\n"+
			"• Endpoint: %s\n"+
			"• IPv6:   %v\n"+
			"• Error:  %s",
		member, checkType, checkName, domain, endpoint, ipv6, errText)

	evID, err := sendText(ctx, body)
	if err != nil {
		log.Log(log.Error, "[matrix] failed to send offline alert: %v", err)
		return
	}

	offlineMap.Store(makeKey(member, checkType, checkName, domain, endpoint, ipv6), evID)
}

func NotifyMemberOnline(
	member, checkType, checkName, domain, endpoint string,
	ipv6 bool,
) {
	if !isReady() {
		return
	}
	key := makeKey(member, checkType, checkName, domain, endpoint, ipv6)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	body := fmt.Sprintf(
		"✅  *ONLINE*\n"+
			"• Member: **%s**\n"+
			"• Check:  %s / %s\n"+
			"• Domain: %s\n"+
			"• Endpoint: %s\n"+
			"• IPv6:   %v",
		member, checkType, checkName, domain, endpoint, ipv6)

	if raw, ok := offlineMap.Load(key); ok {
		if evID, ok2 := raw.(id.EventID); ok2 {
			if editErr := editText(ctx, evID, body); editErr == nil {
				offlineMap.Delete(key)
				return
			} else {
				log.Log(log.Warn, "[matrix] edit failed – falling back to new msg: %v", editErr)
			}
		}
	}

	// Either we had no eventID cached or the edit failed – send a fresh message.
	_, _ = sendText(ctx, body)
}
