package instance

import (
	"fmt"
	"github.com/qpubio/qpub-server/internal/shared/clock"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"time"
)

type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
)

type ServerInstance struct {
	ID id.Int `gorm:"primarykey;autoincrement"`

	InstanceID id.ULID `gorm:"type:char(22);unique;index"`

	// Cluster components
	ClusterScope  string `gorm:"type:varchar(32);index"`
	ClusterEnv    string `gorm:"type:varchar(32);index"`
	ClusterRegion string `gorm:"type:varchar(64);index"`
	ClusterZone   string `gorm:"type:varchar(32);index"`

	// VM components
	VMScope  string `gorm:"type:varchar(32);index"`
	VMEnv    string `gorm:"type:varchar(32);index"`
	VMRegion string `gorm:"type:varchar(64);index"`
	VMZone   string `gorm:"type:varchar(32);index"`
	VMRole   string `gorm:"type:varchar(32);index"`
	VMSeq    string `gorm:"type:varchar(16)"`

	Status Status `gorm:"not null;default:active"`

	CreatedAt time.Time `gorm:"not null;default:now()"`
	UpdatedAt time.Time `gorm:"not null;default:now()"`
}

func (ServerInstance) TableName() string {
	return "server_instances"
}

type CreateParams struct {
	InstanceID id.ULID

	// Cluster components
	ClusterScope  string
	ClusterEnv    string
	ClusterRegion string
	ClusterZone   string

	// VM components
	VMScope  string
	VMEnv    string
	VMRegion string
	VMZone   string
	VMRole   string
	VMSeq    string
}

type UpdateParams struct {
	// Cluster components
	ClusterScope  string
	ClusterEnv    string
	ClusterRegion string
	ClusterZone   string

	// VM components
	VMScope  string
	VMEnv    string
	VMRegion string
	VMZone   string
	VMRole   string
	VMSeq    string

	Status Status
}

func Create(params CreateParams) (*ServerInstance, error) {
	// Validate params
	validator := NewValidator()
	if err := validator.ValidateCreate(params); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	// Create instance
	instance := &ServerInstance{
		InstanceID:    params.InstanceID,
		ClusterScope:  params.ClusterScope,
		ClusterEnv:    params.ClusterEnv,
		ClusterRegion: params.ClusterRegion,
		ClusterZone:   params.ClusterZone,
		VMScope:       params.VMScope,
		VMEnv:         params.VMEnv,
		VMRegion:      params.VMRegion,
		VMZone:        params.VMZone,
		VMRole:        params.VMRole,
		VMSeq:         params.VMSeq,
		Status:        StatusActive,
		CreatedAt:     clock.Now(),
		UpdatedAt:     clock.Now(),
	}

	return instance, nil
}

func (i *ServerInstance) Update(params UpdateParams) error {
	// Validate params
	validator := NewValidator()
	if err := validator.ValidateUpdate(params); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}

	// Update cluster fields
	if params.ClusterScope != "" {
		i.ClusterScope = params.ClusterScope
	}
	if params.ClusterEnv != "" {
		i.ClusterEnv = params.ClusterEnv
	}
	if params.ClusterRegion != "" {
		i.ClusterRegion = params.ClusterRegion
	}
	if params.ClusterZone != "" {
		i.ClusterZone = params.ClusterZone
	}

	// Update VM fields
	if params.VMScope != "" {
		i.VMScope = params.VMScope
	}
	if params.VMEnv != "" {
		i.VMEnv = params.VMEnv
	}
	if params.VMRegion != "" {
		i.VMRegion = params.VMRegion
	}
	if params.VMZone != "" {
		i.VMZone = params.VMZone
	}
	if params.VMRole != "" {
		i.VMRole = params.VMRole
	}
	if params.VMSeq != "" {
		i.VMSeq = params.VMSeq
	}

	// Update status
	if params.Status != "" {
		i.Status = params.Status
	}

	i.UpdatedAt = clock.Now()

	return nil
}

func (s *ServerInstance) IsActive() bool {
	return s.Status == StatusActive
}

// ClusterName returns the reconstructed cluster name in format: scope-env-region+zone
func (s *ServerInstance) ClusterName() string {
	return fmt.Sprintf("%s-%s-%s%s", s.ClusterScope, s.ClusterEnv, s.ClusterRegion, s.ClusterZone)
}

// VMName returns the reconstructed VM name in format: scope-env-region+zone-role-seq
func (s *ServerInstance) VMName() string {
	return fmt.Sprintf("%s-%s-%s%s-%s-%s", s.VMScope, s.VMEnv, s.VMRegion, s.VMZone, s.VMRole, s.VMSeq)
}
