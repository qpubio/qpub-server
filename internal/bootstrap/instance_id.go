package bootstrap

import (
	"fmt"

	"github.com/qpubio/qpub-server/internal/shared/id"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
)

func (a *App) setupInstanceID() error {
	instanceID := id.NewULID()
	if instanceID == "" {
		return fmt.Errorf("failed to generate instance ID")
	}
	a.instanceID = instanceID
	a.logger.Info(log.Instance, "Instance ID: %s%s%s", log.InfoColor, instanceID, "\033[0m")
	return nil
}
