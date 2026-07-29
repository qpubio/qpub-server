package apikey

import (
	"encoding/json"
	"fmt"
	"github.com/qpubio/qpub-server/internal/shared/clock"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"strings"
	"time"

	"gorm.io/gorm"
)

type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
)

// APIKey represents an API key for a project
type APIKey struct {
	ID         id.Int  `gorm:"primarykey;autoincrement"`
	PublicID   id.Hash `gorm:"type:char(11);unique;index"`
	ProjectID  id.Int  `gorm:"not null;index"`
	Name       string
	SecretKey  string          `gorm:"type:text;not null;index"`
	Permission json.RawMessage `gorm:"type:jsonb"`
	Metadata   json.RawMessage `gorm:"type:jsonb"`
	Status     Status          `gorm:"not null;default:'active'"`
	LastUsedAt *time.Time      `gorm:"index"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	ExpiresAt  *time.Time `gorm:"index"`
}

// TableName returns the table name for the APIKey model
func (APIKey) TableName() string {
	return "api_keys"
}

// AfterCreate is a hook that sets the public ID if it's empty
func (ak *APIKey) AfterCreate(tx *gorm.DB) error {
	if ak.PublicID == "" {
		ak.PublicID = id.HashID(ak.ID)
		return tx.Model(ak).Update("public_id", ak.PublicID).Error
	}
	return nil
}

// CreateParams is a struct to hold parameters for creating a new API key
type CreateParams struct {
	ProjectID  id.Int
	Name       string
	Permission json.RawMessage
	Metadata   json.RawMessage
	Status     Status
	ExpiresAt  *time.Time
}

// UpdateParams is a struct to hold parameters for updating an API key
type UpdateParams struct {
	Name       string
	Permission json.RawMessage
	Metadata   json.RawMessage
	Status     Status
	LastUsedAt *time.Time
	ExpiresAt  *time.Time
}

type RequestCreateAPIKey struct {
	Name       string
	Permission json.RawMessage
	Status     Status
	ExpiresAt  *time.Time
}

type RequestUpdateAPIKey struct {
	Name       string
	Permission json.RawMessage
	Status     Status
	ExpiresAt  *time.Time
}

// Create creates a new API key instance with validation and secret key generation and status setting and sanitization
func Create(params CreateParams) (*APIKey, error) {
	// Validate params
	validator := NewValidator()
	if err := validator.ValidateCreate(params); err != nil {
		return nil, err
	}

	if params.Name == "" {
		params.Name = "Unnamed API Key"
	}

	// Set the status to active if it's empty
	if params.Status == "" {
		params.Status = StatusActive
	}

	// Create security service
	security := NewSecurity()

	// Generate secret key
	secretKey, err := security.GenerateSecretKey()
	if err != nil {
		return nil, err
	}

	// Default permission
	if params.Permission == nil {
		params.Permission = json.RawMessage(`{"*":["*"]}`)
	}

	// Create API key instance
	apiKey := &APIKey{
		ProjectID:  params.ProjectID,
		Name:       strings.TrimSpace(params.Name),
		SecretKey:  secretKey,
		Permission: params.Permission,
		Metadata:   params.Metadata,
		Status:     params.Status,
		ExpiresAt:  params.ExpiresAt,
		CreatedAt:  clock.Now(),
		UpdatedAt:  clock.Now(),
	}

	return apiKey, nil
}

// Update updates an API key instance with validation
func (ak *APIKey) Update(params UpdateParams) error {
	// Validate params
	validator := NewValidator()
	if err := validator.ValidateUpdate(params); err != nil {
		return err
	}

	if params.Name != "" {
		ak.Name = strings.TrimSpace(params.Name)
	}
	if params.Permission != nil {
		ak.Permission = params.Permission
	}
	if params.Metadata != nil {
		ak.Metadata = params.Metadata
	}
	if params.Status != "" {
		ak.Status = params.Status
	}
	if params.LastUsedAt != nil {
		ak.LastUsedAt = params.LastUsedAt
	}
	if params.ExpiresAt != nil {
		ak.ExpiresAt = params.ExpiresAt
	}

	ak.UpdatedAt = clock.Now()

	return nil
}

// FullKey returns the full API key
func (ak APIKey) FullKey() string {
	return fmt.Sprintf("%s:%s", ak.PublicID, ak.SecretKey)
}
