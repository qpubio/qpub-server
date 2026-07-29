package dto

import (
	"encoding/json"
	"time"

	"github.com/qpubio/qpub-server/internal/domain/apikey"
	"github.com/qpubio/qpub-server/internal/shared/id"
)

type ProjectAPIKeyDTO struct {
	PublicID   id.Hash         `json:"id"`
	Name       string          `json:"name"`
	FullKey    string          `json:"key"`
	Permission json.RawMessage `json:"permission"`
	Metadata   JSONMetadata    `json:"metadata"`
	Status     apikey.Status   `json:"status"`
	LastUsedAt *time.Time      `json:"last_used_at"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
	ExpiresAt  *time.Time      `json:"expires_at"`
}

func ToProjectAPIKeyDTO(ak *apikey.APIKey) *ProjectAPIKeyDTO {
	return &ProjectAPIKeyDTO{
		PublicID:   ak.PublicID,
		Name:       ak.Name,
		FullKey:    ak.FullKey(),
		Permission: ak.Permission,
		Metadata:   JSONMetadata(ak.Metadata),
		Status:     ak.Status,
		LastUsedAt: ak.LastUsedAt,
		CreatedAt:  ak.CreatedAt,
		UpdatedAt:  ak.UpdatedAt,
		ExpiresAt:  ak.ExpiresAt,
	}
}

func ToProjectAPIKeysDTO(apiKeys []*apikey.APIKey) []*ProjectAPIKeyDTO {
	apiKeyDTOs := make([]*ProjectAPIKeyDTO, len(apiKeys))
	for i, ak := range apiKeys {
		apiKeyDTOs[i] = ToProjectAPIKeyDTO(ak)
	}
	return apiKeyDTOs
}
