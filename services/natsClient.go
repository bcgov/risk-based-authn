package services

import (
	"sync"

	"github.com/nats-io/nats.go"
)

var (
	NatsConn *nats.Conn
	oneNats  sync.Once
)

func ConnectNats(url string) (*nats.Conn, error) {
	var err error
	oneNats.Do(func() {
		NatsConn, err = nats.Connect(url) // "nats://localhost:4222"
	})
	return NatsConn, err
}
