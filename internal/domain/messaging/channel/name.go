package channel

import (
	"fmt"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"strings"
)

// Name is a value object representing a channel name
type Name struct {
	raw       string // The client-facing name (e.g., "abc")
	projectID id.Int // The project ID (e.g., "31")
}

// NewName creates a new channel name
func NewName(rawName string, projID id.Int) Name {
	return Name{
		raw:       rawName,
		projectID: projID,
	}
}

// Raw returns the client-facing channel name
func (n Name) Raw() string {
	return n.raw
}

// Full returns the fully qualified channel name with project prefix
func (n Name) Full() string {
	return fmt.Sprintf("%d.%s", n.projectID, n.raw)
}

// ProjectID returns the project ID associated with this channel
func (n Name) ProjectID() id.Int {
	return n.projectID
}

// FromFull parses a full channel name into a Name object
func FromFull(fullName string) (Name, error) {
	parts := strings.SplitN(fullName, ".", 2)
	if len(parts) != 2 {
		return Name{}, fmt.Errorf("invalid channel name: %s", fullName)
	}

	projID, err := id.ParseInt(parts[0])
	if err != nil {
		return Name{}, fmt.Errorf("invalid channel name: %s", fullName)
	}

	return Name{
		raw:       parts[1],
		projectID: projID,
	}, nil
}
