package lifecycle

import (
	"context"
	"github.com/qpubio/qpub-server/internal/domain/messaging/channel"
	"github.com/qpubio/qpub-server/internal/domain/messaging/event"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
	"sync"
	"time"
)

// CleanupSchedule represents a scheduled cleanup operation
type CleanupSchedule struct {
	ChannelName string
	ProjectID   id.Int
	Timer       *time.Timer
	ScheduledAt time.Time
}

// Manager handles channel lifecycle events and manages delayed cleanup
type Manager struct {
	logger          logger.Service
	channelService  channel.Service
	eventBus        event.Service
	cleanupDelay    time.Duration
	pendingCleanups map[string]*CleanupSchedule // channelFullName -> cleanup schedule
	mutex           sync.RWMutex
	shutdownChan    chan struct{}
	wg              sync.WaitGroup
}

// NewManager creates a new channel lifecycle manager
func NewManager(
	logger logger.Service,
	channelService channel.Service,
	eventBus event.Service,
	cleanupDelay time.Duration,
) *Manager {
	return &Manager{
		logger:          logger,
		channelService:  channelService,
		eventBus:        eventBus,
		cleanupDelay:    cleanupDelay,
		pendingCleanups: make(map[string]*CleanupSchedule),
		shutdownChan:    make(chan struct{}),
	}
}

// Start initializes the manager and subscribes to relevant events
func (m *Manager) Start(ctx context.Context) error {
	m.logger.Info(log.MessagingChannel, `Starting channel lifecycle manager cleanupDelay=%v`, m.cleanupDelay)

	// Subscribe to channel events
	err := m.eventBus.Subscribe(
		event.EventChannelEmpty,
		"channel-lifecycle-manager",
		m.handleChannelEmpty,
	)
	if err != nil {
		return err
	}

	err = m.eventBus.Subscribe(
		event.EventChannelSubscribed,
		"channel-lifecycle-manager",
		m.handleChannelSubscribed,
	)
	if err != nil {
		return err
	}

	m.logger.Info(log.MessagingChannel, "Channel lifecycle manager started")
	return nil
}

// Stop gracefully shuts down the manager
func (m *Manager) Stop(ctx context.Context) error {
	m.logger.Info(log.MessagingChannel, "Stopping channel lifecycle manager")

	// Unsubscribe from events
	m.eventBus.Unsubscribe(event.EventChannelEmpty, "channel-lifecycle-manager")
	m.eventBus.Unsubscribe(event.EventChannelSubscribed, "channel-lifecycle-manager")

	// Cancel all pending cleanups
	m.mutex.Lock()
	for channelName, schedule := range m.pendingCleanups {
		if schedule.Timer != nil {
			schedule.Timer.Stop()
		}
		m.logger.Debug(log.MessagingChannel, `Cancelled pending cleanup channelName=%v`, channelName)
	}
	m.pendingCleanups = make(map[string]*CleanupSchedule)
	m.mutex.Unlock()

	close(m.shutdownChan)
	m.wg.Wait()

	m.logger.Info(log.MessagingChannel, "Channel lifecycle manager stopped")
	return nil
}

// ScheduleCleanup schedules a delayed cleanup for a channel
func (m *Manager) ScheduleCleanup(channelRawName string, projectID id.Int) {
	channelName := channel.NewName(channelRawName, projectID)
	fullName := channelName.Full()

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Cancel existing timer if any
	if existingSchedule, exists := m.pendingCleanups[fullName]; exists {
		if existingSchedule.Timer != nil {
			existingSchedule.Timer.Stop()
		}
		m.logger.Debug(log.MessagingChannel, `Cancelled existing cleanup schedule channelName=%v projectID=%v`, channelRawName,
			projectID)
	}

	// Schedule new cleanup
	timer := time.AfterFunc(m.cleanupDelay, func() {
		m.performCleanup(channelRawName, projectID)
	})

	schedule := &CleanupSchedule{
		ChannelName: channelRawName,
		ProjectID:   projectID,
		Timer:       timer,
		ScheduledAt: time.Now(),
	}

	m.pendingCleanups[fullName] = schedule

	m.logger.Info(log.MessagingChannel, `Scheduled channel cleanup channelName=%v projectID=%v cleanupDelay=%v scheduledAt=%v`, channelRawName,
		projectID,
		m.cleanupDelay,
		schedule.ScheduledAt)
}

// CancelCleanup cancels a scheduled cleanup for a channel
func (m *Manager) CancelCleanup(channelRawName string, projectID id.Int) {
	channelName := channel.NewName(channelRawName, projectID)
	fullName := channelName.Full()

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if schedule, exists := m.pendingCleanups[fullName]; exists {
		if schedule.Timer != nil {
			schedule.Timer.Stop()
		}
		delete(m.pendingCleanups, fullName)

		m.logger.Info(log.MessagingChannel, `Cancelled channel cleanup channelName=%v projectID=%v wasScheduledAt=%v`, channelRawName,
			projectID,
			schedule.ScheduledAt)
	}
}

// GetPendingCleanups returns information about pending cleanups
func (m *Manager) GetPendingCleanups() map[string]CleanupSchedule {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make(map[string]CleanupSchedule)
	for fullName, schedule := range m.pendingCleanups {
		result[fullName] = *schedule
	}

	return result
}

// performCleanup actually performs the channel cleanup
func (m *Manager) performCleanup(channelRawName string, projectID id.Int) {
	channelName := channel.NewName(channelRawName, projectID)
	fullName := channelName.Full()

	m.logger.Info(log.MessagingChannel, `Performing delayed channel cleanup channelName=%v projectID=%v`, channelRawName,
		projectID)

	// Remove from pending cleanups first
	m.mutex.Lock()
	delete(m.pendingCleanups, fullName)
	m.mutex.Unlock()

	// Check if channel still exists and has no subscribers
	ch, err := m.channelService.Get(channelRawName, projectID)
	if err != nil {
		if err == channel.ErrNotFound {
			m.logger.Debug(log.MessagingChannel, `Channel already deleted channelName=%v projectID=%v`, channelRawName,
				projectID)
		} else {
			m.logger.Error(log.MessagingChannel, `Error getting channel for cleanup channelName=%v projectID=%v error=%v`, channelRawName,
				projectID,
				err)
		}
		return
	}

	// Double-check that channel has no subscribers
	if ch.HasLocalSubscribers() {
		m.logger.Info(log.MessagingChannel, `Channel gained subscribers during cleanup delay, skipping deletion channelName=%v projectID=%v subscriberCount=%v`, channelRawName,
			projectID,
			ch.LocalSubscriptionCount())
		return
	}

	// Delete the channel
	err = m.channelService.Delete(channelRawName, projectID)
	if err != nil {
		m.logger.Error(log.MessagingChannel, `Failed to delete channel during cleanup channelName=%v projectID=%v error=%v`, channelRawName,
			projectID,
			err)
		return
	}

	// Publish channel deleted event
	evt := event.NewEvent(event.EventChannelDeleted, event.ChannelDeletedData{
		ChannelName: channelRawName,
		ProjectID:   projectID,
		InstanceID:  ch.InstanceID(),
	})

	if err := m.eventBus.Publish(context.Background(), evt); err != nil {
		m.logger.Error(log.MessagingChannel, `Failed to publish channel deleted event channelName=%v projectID=%v error=%v`, channelRawName,
			projectID,
			err)
	}

	m.logger.Info(log.MessagingChannel, `Channel successfully deleted during cleanup channelName=%v projectID=%v`, channelRawName,
		projectID)
}

// handleChannelEmpty handles the channel empty event
func (m *Manager) handleChannelEmpty(evt *event.Event) error {
	data, ok := evt.Data.(event.ChannelEmptyData)
	if !ok {
		return nil // Ignore invalid event data
	}

	m.logger.Debug(log.MessagingChannel, `Handling channel empty event channelName=%v projectID=%v`, data.ChannelName,
		data.ProjectID)

	m.ScheduleCleanup(data.ChannelName, data.ProjectID)
	return nil
}

// handleChannelSubscribed handles the channel subscribed event
func (m *Manager) handleChannelSubscribed(evt *event.Event) error {
	data, ok := evt.Data.(event.ChannelSubscribedData)
	if !ok {
		return nil // Ignore invalid event data
	}

	m.logger.Debug(log.MessagingChannel, `Handling channel subscribed event channelName=%v projectID=%v`, data.ChannelName,
		data.ProjectID)

	// Cancel any pending cleanup for this channel
	m.CancelCleanup(data.ChannelName, data.ProjectID)
	return nil
}
