package apikey

import "github.com/qpubio/qpub-server/internal/domain/apikey"

type Service interface {
	Authenticate(apiKeyString string) (*apikey.APIKey, error)
}
