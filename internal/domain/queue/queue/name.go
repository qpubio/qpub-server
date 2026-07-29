package queue

import (
	"fmt"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"strings"
)

// Name is a value object representing a queue name scoped to a project.
type Name struct {
	raw       string
	projectID id.Int
}

// NewName creates a queue name for a project.
func NewName(rawName string, projectID id.Int) Name {
	return Name{
		raw:       rawName,
		projectID: projectID,
	}
}

// Raw returns the client-facing queue name.
func (n Name) Raw() string {
	return n.raw
}

// ProjectID returns the owning project ID.
func (n Name) ProjectID() id.Int {
	return n.projectID
}

// Full returns the fully qualified queue name.
func (n Name) Full() string {
	return fmt.Sprintf("%d.%s", n.projectID, n.raw)
}

// Subject returns the JetStream subject for this queue.
func (n Name) Subject() string {
	return fmt.Sprintf("qpub.jobs.%d.%s", n.projectID, n.raw)
}

// DLQSubject returns the dead-letter JetStream subject.
func (n Name) DLQSubject() string {
	return fmt.Sprintf("qpub.dlq.%d.%s", n.projectID, n.raw)
}

// FromFull parses a fully qualified queue name.
func FromFull(fullName string) (Name, error) {
	parts := strings.SplitN(fullName, ".", 2)
	if len(parts) != 2 {
		return Name{}, fmt.Errorf("invalid queue name: %s", fullName)
	}

	projectID, err := id.ParseInt(parts[0])
	if err != nil {
		return Name{}, fmt.Errorf("invalid queue name: %s", fullName)
	}

	return Name{
		raw:       parts[1],
		projectID: projectID,
	}, nil
}
