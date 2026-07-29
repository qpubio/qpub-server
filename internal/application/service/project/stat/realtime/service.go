package realtime

import (
	"github.com/qpubio/qpub-server/internal/domain/project/stat/realtime"
	"github.com/qpubio/qpub-server/internal/shared/id"
)

type Service struct {
	repository realtime.Repository
}

func NewService(repository realtime.Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Incr(key realtime.Key) error {
	return s.repository.Incr(key)
}

func (s *Service) IncrBy(key realtime.Key, value int64) error {
	return s.repository.IncrBy(key, value)
}

func (s *Service) Decr(key realtime.Key) error {
	return s.repository.Decr(key)
}

func (s *Service) DecrBy(key realtime.Key, value int64) error {
	return s.repository.DecrBy(key, value)
}

func (s *Service) Get(key realtime.Key) (int64, error) {
	return s.repository.Get(key)
}

func (s *Service) Set(key realtime.Key, value int64) error {
	return s.repository.Set(key, value)
}

func (s *Service) GetByPattern(pattern string) (map[string]int64, error) {
	keys, err := s.repository.GetByPattern(pattern)
	if err != nil {
		return nil, err
	}

	result := make(map[string]int64)
	for _, keyStr := range keys {
		key, err := realtime.ParseKey(keyStr)
		if err != nil || key == nil {
			continue // Skip invalid keys
		}

		value, err := s.Get(*key)
		if err != nil {
			continue // Skip keys with errors
		}

		result[keyStr] = value
	}

	return result, nil
}

func (s *Service) Reset(key realtime.Key) error {
	return s.repository.Reset(key)
}

func (s *Service) ResetByPattern(pattern string) error {
	return s.repository.ResetByPattern(pattern)
}

func (s *Service) GetSummary(projID id.Int) (map[string]int64, error) {
	summary := make(map[string]int64)

	keyTypes := []realtime.KeyType{
		realtime.KeyConnection,
		realtime.KeyChannel,
		realtime.KeySubscriber,
		realtime.KeyMessageInbound,
		realtime.KeyMessageOutbound,
		realtime.KeyMessageDropped,
		realtime.KeyBandwidthInbound,
		realtime.KeyBandwidthOutbound,
	}

	// Initialize all counters to zero
	for _, keyType := range keyTypes {
		summary[string(keyType)] = 0

		// Create key for this type with empty instance ID
		key := realtime.NewKey(keyType, "", projID)

		// Use the proper Pattern() method
		pattern := key.Pattern()

		// Get stats using the service's GetByPattern method
		stats, err := s.GetByPattern(pattern)
		if err != nil {
			continue
		}

		// Sum all values for this key type
		for _, val := range stats {
			summary[string(keyType)] += val
		}
	}

	return summary, nil
}
