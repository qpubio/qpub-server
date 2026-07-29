package realtime

import (
	"fmt"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"strings"
)

// KeyType represents the type of realtime statistic being tracked
type KeyType string

// KeyType constants
const (
	KeyConnection       KeyType = "conn"
	KeyChannel          KeyType = "chan"
	KeySubscriber       KeyType = "sub"
	KeyMessageInbound   KeyType = "msg:in"
	KeyMessageOutbound  KeyType = "msg:out"
	KeyMessageDropped   KeyType = "msg:drop"
	KeyBandwidthInbound KeyType = "bw:in"
	KeyBandwidthOutbound KeyType = "bw:out"
)

// Key represents a realtime statistic key
type Key struct {
	Prefix     string
	Separator  string
	Type       KeyType
	InstanceID id.ULID
	ProjectID  id.Int
}

// NewKey creates a new Key instance
func NewKey(keyType KeyType, instanceID id.ULID, projID id.Int) *Key {
	return &Key{
		Prefix:     "stats",
		Separator:  ":",
		Type:       keyType,
		InstanceID: instanceID,
		ProjectID:  projID,
	}
}

// String returns the string representation of the key
func (k *Key) String() string {
	return fmt.Sprintf("%s%s%s%s%s%s%d",
		k.Prefix, k.Separator,
		k.Type, k.Separator,
		k.InstanceID, k.Separator,
		k.ProjectID)
}

// Pattern returns the pattern for the key based on provided components
func (k *Key) Pattern() string {
	parts := []string{k.Prefix}

	// Add type if present
	if k.Type != "" {
		parts = append(parts, string(k.Type))
	} else {
		parts = append(parts, "*")
	}

	// Add instance ID if present
	if k.InstanceID != "" {
		parts = append(parts, k.InstanceID)
	} else {
		parts = append(parts, "*")
	}

	// Add project ID if present
	if k.ProjectID != 0 {
		parts = append(parts, fmt.Sprint(k.ProjectID))
	} else {
		parts = append(parts, "*")
	}

	return strings.Join(parts, k.Separator)
}

// ParseKey creates a Key from a string representation
func ParseKey(keyStr string) (*Key, error) {
	parts := strings.Split(keyStr, ":")
	if len(parts) < 4 { // Need at least 4 parts
		return nil, fmt.Errorf("invalid key format: %s", keyStr)
	}

	// The first part is always "stats"
	// prefix := parts[0]

	// The last part is always projectID
	lastIndex := len(parts) - 1
	projectIDStr := parts[lastIndex]
	projectID, err := id.ParseInt(projectIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %s", projectIDStr)
	}

	// Second to last part is always instanceID
	instanceID := id.ULID(parts[lastIndex-1])

	// Everything between prefix and instanceID is the key type
	// This handles composite types like "msg:sent"
	typeComponents := parts[1 : lastIndex-1]
	keyType := KeyType(strings.Join(typeComponents, ":"))

	return NewKey(keyType, instanceID, projectID), nil
}

// Stat represents a realtime stat with its value
type Stat struct {
	Key   Key
	Value int64
}

// Create creates a new Stat instance
func Create(key Key, value int64) *Stat {
	return &Stat{
		Key:   key,
		Value: value,
	}
}
