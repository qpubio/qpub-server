package delivery

import "github.com/qpubio/qpub-server/internal/shared/id"

// Deliverer enqueues outbound payloads for a client connection.
type Deliverer interface {
	Deliver(clientID id.ULID, payload []byte) error
}
