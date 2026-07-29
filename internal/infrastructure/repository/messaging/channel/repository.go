package channel

import (
	"github.com/qpubio/qpub-server/internal/domain/messaging/channel"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
	"sync"
)

type repository struct {
	logger   logger.Service
	channels map[string]*channel.Channel // channelName -> channel object
	mu       sync.RWMutex
}

func NewRepository(logger logger.Service) channel.Repository {
	return &repository{
		logger:   logger,
		channels: make(map[string]*channel.Channel),
	}
}

func (r *repository) Create(ch *channel.Channel) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.channels[ch.FullName()]; exists {
		return channel.ErrAlreadyExists
	}

	r.channels[ch.FullName()] = ch
	r.logger.Debug(log.MessagingChannel, `Channel created in repository channel=%v fullName=%v`, ch.RawName(),
		ch.FullName())

	return nil
}

func (r *repository) Update(ch *channel.Channel) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.channels[ch.FullName()]; !exists {
		return channel.ErrNotFound
	}

	r.channels[ch.FullName()] = ch
	r.logger.Debug(log.MessagingChannel, `Channel updated in repository channel=%v fullName=%v`, ch.RawName(),
		ch.FullName())

	return nil
}

func (r *repository) Delete(ch *channel.Channel) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.channels[ch.FullName()]; !exists {
		return channel.ErrNotFound
	}

	delete(r.channels, ch.FullName())
	r.logger.Debug(log.MessagingChannel, `Channel deleted from repository channel=%v fullName=%v`, ch.RawName(),
		ch.FullName())

	return nil
}

func (r *repository) FindByName(fullName string) (*channel.Channel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ch, ok := r.channels[fullName]
	if !ok {
		return nil, channel.ErrNotFound
	}

	return ch, nil
}

func (r *repository) FindAllLocal() ([]*channel.Channel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	channels := make([]*channel.Channel, 0, len(r.channels))
	for _, ch := range r.channels {
		channels = append(channels, ch)
	}

	return channels, nil
}

// LogChannels logs all channels in the repository (mainly for debugging)
func (r *repository) LogChannels() {
	r.mu.RLock()
	defer r.mu.RUnlock()

	r.logger.Info(log.MessagingChannel, `Current local channels count=%v`, len(r.channels))

	for fullName, ch := range r.channels {
		r.logger.Info(log.MessagingChannel, `Channel details fullName=%v rawName=%v active=%v subscribers=%v`, fullName,
			ch.RawName(),
			ch.IsActive(),
			ch.LocalSubscriptionCount())
	}
}
