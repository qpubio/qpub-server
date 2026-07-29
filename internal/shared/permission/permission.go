package permission

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Action represents the action that can be performed on a resource
type Action string

// Action constants
const (
	ActionPublish   Action = "publish"
	ActionSubscribe Action = "subscribe"
	ActionStats     Action = "stats"
	ActionLogs      Action = "logs"
	ActionEnqueue   Action = "enqueue"
	ActionDequeue   Action = "dequeue"
	ActionAll       Action = "*"
)

// Platform channel names reserved for project telemetry.
const (
	ChannelStats = "_stats"
	ChannelLogs  = "_logs"
)

// Permission represents a set of permissions for resources
type Permission struct {
	Resources map[string][]Action
}

// New creates a new Permission instance
func New(resources map[string][]Action) *Permission {
	if resources == nil {
		resources = make(map[string][]Action)
	}
	return &Permission{
		Resources: resources,
	}
}

// FromJSON creates a Permission instance from JSON bytes
func FromJSON(data []byte) (*Permission, error) {
	if len(data) == 0 {
		return New(nil), nil
	}

	// Try to unmarshal the new format first (direct map of resource to actions)
	resourceMap := make(map[string][]Action)
	if err := json.Unmarshal(data, &resourceMap); err == nil {
		return New(resourceMap), nil
	}

	// If that fails, try the old format with the "resources" wrapper
	var p struct {
		Resources map[string][]Action `json:"resources"`
	}
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("failed to unmarshal permission: %w", err)
	}
	return New(p.Resources), nil
}

// HasData checks if the permission JSON contains meaningful data
// Returns false for nil, empty, or empty JSON structures ({}, [], null)
func HasData(data []byte) bool {
	if len(data) == 0 {
		return false
	}

	// Trim whitespace
	trimmed := bytes.TrimSpace(data)

	// Check for empty JSON values
	if bytes.Equal(trimmed, []byte("{}")) ||
		bytes.Equal(trimmed, []byte("null")) ||
		bytes.Equal(trimmed, []byte("[]")) {
		return false
	}

	return true
}

// Can checks if the permission allows a specific action on a resource
func (p *Permission) Can(resource string, action Action) bool {
	if p.Resources == nil {
		return false
	}

	// Check wildcard resource permissions
	if actions, exists := p.Resources["*"]; exists {
		// If resource is "*" and it contains "*" action, grant all permissions
		if containsAction(actions, ActionAll) {
			return true
		}
		// Otherwise check if the specific action is allowed
		if containsAction(actions, action) {
			return true
		}
	}

	// Check specific resource permissions
	if actions, exists := p.Resources[resource]; exists {
		if containsAction(actions, action) {
			return true
		}
	}

	return false
}

// containsAction checks if the action list contains the target action or wildcard
func containsAction(actions []Action, targetAction Action) bool {
	for _, action := range actions {
		if action == ActionAll || action == targetAction {
			return true
		}
	}
	return false
}

// ToJSON converts the Permission to JSON bytes
func (p *Permission) ToJSON() ([]byte, error) {
	return json.Marshal(p.Resources)
}
