package nats

import (
	"sync"

	"github.com/nats-io/nats.go"
)

var (
	nc     *nats.Conn
	natsMu sync.Mutex
)
