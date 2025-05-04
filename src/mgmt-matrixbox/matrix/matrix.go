package matrix

import (
	"context"
	"fmt"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

// NewMatrixBot creates and logs in a new Matrix bot.
func NewMatrixBot(homeserverURL, username, password, roomID string) (*MatrixBot, error) {
	client, err := mautrix.NewClient(homeserverURL, "", "")
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	loginReq := mautrix.ReqLogin{
		Type: mautrix.AuthTypePassword,
		Identifier: mautrix.UserIdentifier{
			Type: mautrix.IdentifierTypeUser,
			User: username,
		},
		Password: password,
	}

	loginResp, err := client.Login(context.Background(), &loginReq)
	if err != nil {
		return nil, fmt.Errorf("failed to log in: %w", err)
	}

	client.SetCredentials(loginResp.UserID, loginResp.AccessToken)

	return &MatrixBot{
		Client: client,
		RoomID: id.RoomID(roomID),
	}, nil
}

// SendMessage sends an HTML-formatted message to the Matrix room.
func (bot *MatrixBot) SendMessage(htmlMessage string) error {
	content := event.MessageEventContent{
		MsgType:       event.MsgText,
		Format:        "org.matrix.custom.html",
		FormattedBody: htmlMessage,
	}

	_, err := bot.Client.SendMessageEvent(context.Background(), bot.RoomID, event.EventMessage, content)
	if err != nil {
		return fmt.Errorf("failed to send formatted message: %w", err)
	}

	return nil
}
