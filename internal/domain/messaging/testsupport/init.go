package testsupport

import (
	sharedcfg "github.com/qpubio/qpub-server/internal/config/shared"
	"github.com/qpubio/qpub-server/internal/shared/clock"
	"github.com/qpubio/qpub-server/internal/shared/id"
)

// InitRuntime initializes shared services required by messaging domain tests.
func InitRuntime() {
	clock.Init()
	cfg := &sharedcfg.ID{
		HashSalt:   "test-salt-for-messaging-domain",
		HashLength: 8,
		ULIDLength: 22,
	}
	_ = id.Init(cfg)
}
