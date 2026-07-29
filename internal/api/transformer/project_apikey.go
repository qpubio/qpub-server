package transformer

import (
	appDTO "github.com/qpubio/qpub-server/internal/application/dto"
	"github.com/qpubio/qpub-server/internal/domain/apikey"
)

func TransformProjectAPIKey(ak *apikey.APIKey) *appDTO.ProjectAPIKeyDTO {
	dto := appDTO.ToProjectAPIKeyDTO(ak)
	return dto
}

func TransformProjectAPIKeys(apiKeys []apikey.APIKey) []*appDTO.ProjectAPIKeyDTO {
	apiKeysPtrs := make([]*apikey.APIKey, len(apiKeys))
	for i, ak := range apiKeys {
		apiKeysPtrs[i] = &ak
	}
	dto := appDTO.ToProjectAPIKeysDTO(apiKeysPtrs)
	return dto
}
