package claim

import (
	"encoding/json"
	"github.com/qpubio/qpub-server/internal/shared/permission"
)

// Permission handles permission checks for claims
type Permission struct {
	service permission.Service
}

// NewPermission creates a new Permission instance
func NewPermission() *Permission {
	return &Permission{
		service: permission.NewService(),
	}
}

// CanPublish checks if the permission data allows publishing to a resource
func (p *Permission) CanPublish(data json.RawMessage, resource string) (bool, error) {
	return p.service.CanPublish(data, resource)
}

// CanSubscribe checks if the permission data allows subscribing to a resource
func (p *Permission) CanSubscribe(data json.RawMessage, resource string) (bool, error) {
	return p.service.CanSubscribe(data, resource)
}

// CanStats checks if the permission data allows viewing stats for a resource
func (p *Permission) CanStats(data json.RawMessage, resource string) (bool, error) {
	return p.service.CanStats(data, resource)
}

// CanLogs checks if the permission data allows viewing logs for a resource
func (p *Permission) CanLogs(data json.RawMessage, resource string) (bool, error) {
	return p.service.CanLogs(data, resource)
}
