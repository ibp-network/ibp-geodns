package matrix

import (
	"context"
	"fmt"
	"strings"
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
	userID     id.UserID       // local Matrix user (after login)
	roomID     id.RoomID       // destination room to post to
	once       sync.Once       // protect Init()
	offlineMap sync.Map        // outage‑key → id.EventID   (for edits & deduplication)
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

func getMemberMentions(memberName string) []string {
	c := cfg.GetConfig()

	memberKey := strings.ToLower(memberName)
	if users, ok := c.Alerts.Matrix.Members[memberKey]; ok {
		return users
	}

	return nil
}

// formatAlert creates both plain text and HTML versions of an alert message.
func formatAlert(isOffline bool, member, checkType, checkName, domain, endpoint string, ipv6 bool, errText string, mentions []string) (body, html string) {
	// Build mention prefix if needed
	mentionText := ""
	mentionHTML := ""
	if len(mentions) > 0 {
		mentionText = strings.Join(mentions, " ") + "\n"
		mentionHTML = strings.Join(mentions, " ") + "<br/>"
	}

	// Common fields for both online and offline
	status := "✅  *ONLINE*"
	statusHTML := "✅  <strong>ONLINE</strong>"
	fields := fmt.Sprintf(
		"• Member: **%s**\n"+
			"• Check:  %s / %s\n"+
			"• Domain: %s\n"+
			"• Endpoint: %s\n"+
			"• IPv6:   %v",
		member, checkType, checkName, domain, endpoint, ipv6)
	fieldsHTML := fmt.Sprintf(
		"• Member: <strong>%s</strong><br/>"+
			"• Check:  %s / %s<br/>"+
			"• Domain: %s<br/>"+
			"• Endpoint: %s<br/>"+
			"• IPv6:   %v",
		member, checkType, checkName, domain, endpoint, ipv6)

	// Add offline-specific fields
	if isOffline {
		status = "⚠️  *OFFLINE*"
		statusHTML = "⚠️  <strong>OFFLINE</strong>"
		fields += fmt.Sprintf("\n• Error:  %s", errText)
		fieldsHTML += fmt.Sprintf("<br/>• Error:  %s", errText)
	}

	body = mentionText + status + "\n" + fields
	html = mentionHTML + statusHTML + "<br/>" + fieldsHTML

	return body, html
}

// sendFormattedText posts an HTML formatted message.
func sendFormattedText(ctx context.Context, body, formattedBody string) (id.EventID, error) {
	content := map[string]interface{}{
		"msgtype":        "m.text",
		"body":           body,
		"format":         "org.matrix.custom.html",
		"formatted_body": formattedBody,
	}

	resp, err := client.SendMessageEvent(ctx, roomID, event.EventMessage, content)
	if err != nil {
		return "", err
	}
	return resp.EventID, nil
}

// editFormattedText performs an *in‑place* edit with HTML content.
func editFormattedText(ctx context.Context, target id.EventID, body, formattedBody string) error {
	content := map[string]interface{}{
		"msgtype":        "m.text",
		"body":           body,
		"format":         "org.matrix.custom.html",
		"formatted_body": formattedBody,
		"m.new_content": map[string]interface{}{
			"msgtype":        "m.text",
			"body":           body,
			"format":         "org.matrix.custom.html",
			"formatted_body": formattedBody,
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

// NotifyMemberOffline posts a single alert for a given outage, regardless of
// how many times the caller tries to report it.
func NotifyMemberOffline(
	member, checkType, checkName, domain, endpoint string,
	ipv6 bool, errText string,
) {
	if !isReady() {
		return
	}

	key := makeKey(member, checkType, checkName, domain, endpoint, ipv6)

	// ---------------------------------------------------------------------
	// DEDUPLICATION LOGIC
	// ---------------------------------------------------------------------
	sentinel := id.EventID("")
	if prev, loaded := offlineMap.LoadOrStore(key, sentinel); loaded {
		if prev.(id.EventID) != "" {
			// Already announced.
			return
		}
		// Another goroutine is announcing – skip duplicate.
		return
	}

	//----------------------------------------------------------------------
	// We are the "announcer" for this outage.
	//----------------------------------------------------------------------
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get member mentions and format message
	mentions := getMemberMentions(member)
	body, formattedBody := formatAlert(true, member, checkType, checkName, domain, endpoint, ipv6, errText, mentions)

	evID, err := sendFormattedText(ctx, body, formattedBody)
	if err != nil {
		// Clean‑up sentinel so future attempts can retry.
		offlineMap.Delete(key)
		log.Log(log.Error, "[matrix] failed to send offline alert: %v", err)
		return
	}

	offlineMap.Store(key, evID)
}

// NotifyMemberOnline edits the existing alert back to *ONLINE* status.  If the
// original alert is missing or the edit fails, it falls back to sending a new
// message.
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

	// Format message (no mentions for online alerts)
	body, formattedBody := formatAlert(false, member, checkType, checkName, domain, endpoint, ipv6, "", nil)

	if raw, ok := offlineMap.Load(key); ok {
		if evID, ok2 := raw.(id.EventID); ok2 && evID != "" {
			// Attempt edit‑in‑place.
			editErr := editFormattedText(ctx, evID, body, formattedBody)
			if editErr == nil {
				offlineMap.Delete(key)
				return
			}
			log.Log(log.Warn, "[matrix] edit failed – falling back to new msg: %v", editErr)
		}
	}

	// Either we had no cached event or the edit did not work – send a fresh one.
	_, _ = sendFormattedText(ctx, body, formattedBody)
	offlineMap.Delete(key) // ensure future OFFLINE alerts are allowed again
}
