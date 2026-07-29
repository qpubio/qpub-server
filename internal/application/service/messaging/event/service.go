package event

import (
	"context"
	"fmt"
	"github.com/qpubio/qpub-server/internal/domain/messaging/event"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
	"sync"
)

// Service implements the event bus service
type Service struct {
	logger   logger.Service
	handlers map[event.EventType]map[string]*event.HandlerRegistration
	mutex    sync.RWMutex
	closed   bool
}

// NewService creates a new event bus service
func NewService(logger logger.Service) event.Service {
	return &Service{
		logger:   logger,
		handlers: make(map[event.EventType]map[string]*event.HandlerRegistration),
	}
}

// Publish publishes an event asynchronously to all registered handlers
func (s *Service) Publish(ctx context.Context, evt *event.Event) error {
	if s.closed {
		return fmt.Errorf("event bus is closed")
	}

	handlers := s.getHandlersForEvent(evt.Type)
	if len(handlers) == 0 {
		s.logger.Debug(log.App, `No handlers registered for event type eventType=%v eventID=%v`, evt.Type,
			evt.ID)
		return nil
	}

	// Publish asynchronously
	go s.publishToHandlers(ctx, evt, handlers)

	return nil
}

// PublishSync publishes an event synchronously to all registered handlers
func (s *Service) PublishSync(ctx context.Context, evt *event.Event) error {
	if s.closed {
		return fmt.Errorf("event bus is closed")
	}

	handlers := s.getHandlersForEvent(evt.Type)
	if len(handlers) == 0 {
		s.logger.Debug(log.App, `No handlers registered for event type eventType=%v eventID=%v`, evt.Type,
			evt.ID)
		return nil
	}

	return s.publishToHandlers(ctx, evt, handlers)
}

// Subscribe registers a handler for a specific event type
func (s *Service) Subscribe(eventType event.EventType, handlerID string, handler event.EventHandler) error {
	return s.SubscribeWithDescription(eventType, handlerID, handler, "")
}

// SubscribeWithDescription registers a handler with a description
func (s *Service) SubscribeWithDescription(eventType event.EventType, handlerID string, handler event.EventHandler, description string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.closed {
		return fmt.Errorf("event bus is closed")
	}

	// Initialize map for event type if it doesn't exist
	if _, exists := s.handlers[eventType]; !exists {
		s.handlers[eventType] = make(map[string]*event.HandlerRegistration)
	}

	// Check if handler ID already exists
	if _, exists := s.handlers[eventType][handlerID]; exists {
		return fmt.Errorf("handler with ID %s already registered for event type %s", handlerID, eventType)
	}

	// Register the handler
	registration := &event.HandlerRegistration{
		ID:          handlerID,
		EventType:   eventType,
		Handler:     handler,
		Description: description,
	}

	s.handlers[eventType][handlerID] = registration

	s.logger.Info(log.App, `Event handler registered eventType=%v handlerID=%v description=%v`, eventType,
		handlerID,
		description)

	return nil
}

// Unsubscribe removes a handler for a specific event type
func (s *Service) Unsubscribe(eventType event.EventType, handlerID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.closed {
		return fmt.Errorf("event bus is closed")
	}

	eventHandlers, exists := s.handlers[eventType]
	if !exists {
		return fmt.Errorf("no handlers registered for event type %s", eventType)
	}

	if _, exists := eventHandlers[handlerID]; !exists {
		return fmt.Errorf("handler with ID %s not found for event type %s", handlerID, eventType)
	}

	delete(eventHandlers, handlerID)

	// Clean up empty event type map
	if len(eventHandlers) == 0 {
		delete(s.handlers, eventType)
	}

	s.logger.Info(log.App, `Event handler unregistered eventType=%v handlerID=%v`, eventType,
		handlerID)

	return nil
}

// GetHandlers returns all registered handlers for an event type
func (s *Service) GetHandlers(eventType event.EventType) []event.HandlerRegistration {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	eventHandlers, exists := s.handlers[eventType]
	if !exists {
		return []event.HandlerRegistration{}
	}

	registrations := make([]event.HandlerRegistration, 0, len(eventHandlers))
	for _, registration := range eventHandlers {
		registrations = append(registrations, *registration)
	}

	return registrations
}

// GetAllHandlers returns all registered handlers
func (s *Service) GetAllHandlers() map[event.EventType][]event.HandlerRegistration {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	result := make(map[event.EventType][]event.HandlerRegistration)

	for eventType, eventHandlers := range s.handlers {
		registrations := make([]event.HandlerRegistration, 0, len(eventHandlers))
		for _, registration := range eventHandlers {
			registrations = append(registrations, *registration)
		}
		result[eventType] = registrations
	}

	return result
}

// Close gracefully shuts down the event bus
func (s *Service) Close(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true
	s.handlers = make(map[event.EventType]map[string]*event.HandlerRegistration)

	s.logger.Info(log.App, "Event bus closed")
	return nil
}

// getHandlersForEvent returns handlers for a specific event type (thread-safe)
func (s *Service) getHandlersForEvent(eventType event.EventType) []*event.HandlerRegistration {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	eventHandlers, exists := s.handlers[eventType]
	if !exists {
		return []*event.HandlerRegistration{}
	}

	handlers := make([]*event.HandlerRegistration, 0, len(eventHandlers))
	for _, registration := range eventHandlers {
		handlers = append(handlers, registration)
	}

	return handlers
}

// publishToHandlers publishes an event to all provided handlers
func (s *Service) publishToHandlers(ctx context.Context, evt *event.Event, handlers []*event.HandlerRegistration) error {
	_ = ctx // Context reserved for future timeout/cancellation support

	s.logger.Debug(log.App, `Publishing event to handlers eventType=%v eventID=%v handlerCount=%v`, evt.Type,
		evt.ID,
		len(handlers))

	var errors []error

	for _, registration := range handlers {
		func() {
			defer func() {
				if r := recover(); r != nil {
					s.logger.Error(log.App, `Event handler panicked eventType=%v eventID=%v handlerID=%v panic=%v`, evt.Type,
						evt.ID,
						registration.ID,
						r)
				}
			}()

			if err := registration.Handler(evt); err != nil {
				s.logger.Error(log.App, `Event handler returned error eventType=%v eventID=%v handlerID=%v error=%v`, evt.Type,
					evt.ID,
					registration.ID,
					err)
				errors = append(errors, fmt.Errorf("handler %s: %w", registration.ID, err))
			} else {
				s.logger.Debug(log.App, `Event handler completed successfully eventType=%v eventID=%v handlerID=%v`, evt.Type,
					evt.ID,
					registration.ID)
			}
		}()
	}

	if len(errors) > 0 {
		return fmt.Errorf("event handling errors: %v", errors)
	}

	return nil
}
