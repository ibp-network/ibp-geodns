package matrix

import (
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/id"
)

// MatrixBot represents a Matrix client and the room it operates in.
type MatrixBot struct {
	Client *mautrix.Client
	RoomID id.RoomID
}
