package apikey

import (
	"errors"
	authAPIKey "github.com/qpubio/qpub-server/internal/domain/auth/apikey"
	domainAPIKey "github.com/qpubio/qpub-server/internal/domain/apikey"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	sharedAPIKey "github.com/qpubio/qpub-server/internal/shared/apikey"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
)

type Service struct {
	logger        logger.Service
	apiKeyService domainAPIKey.Service
	apiKeyParser  *sharedAPIKey.Parser
}

func NewService(
	logger logger.Service,
	apiKeyService domainAPIKey.Service,
	apiKeyParser *sharedAPIKey.Parser,
) authAPIKey.Service {
	return &Service{
		logger:        logger,
		apiKeyService: apiKeyService,
		apiKeyParser:  apiKeyParser,
	}
}

func (s *Service) Authenticate(apiKeyString string) (*domainAPIKey.APIKey, error) {
	// Parse the API key to extract components
	apiKeyID, secretKey, err := s.apiKeyParser.ParseKey(apiKeyString)
	if err != nil {
		s.logger.Error(log.AuthAPIKey, "Failed to parse API key: %v", err)
		return nil, errors.New("invalid API key format")
	}

	// Get the API key from the database
	storedAPIKey, err := s.apiKeyService.Get(apiKeyID)
	if err != nil {
		s.logger.Error(log.AuthAPIKey, "Failed to retrieve API key: %v", err)
		return nil, errors.New("API key not found")
	}

	// Verify the secret key
	if storedAPIKey.SecretKey != secretKey {
		s.logger.Warn(log.AuthAPIKey, "Invalid API key secret for keyID %v", apiKeyID)
		return nil, errors.New("invalid API key credentials")
	}

	return &storedAPIKey, nil
}
