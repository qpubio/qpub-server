package websocket

import (
	"github.com/qpubio/qpub-server/internal/domain/messaging/connection"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/clock"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
	"sync"
	"time"
)

// Server represents a WebSocket server that manages WebSocket operations
type Server struct {
	logger              logger.Service
	connectionService   connection.Service
	shutdown            chan struct{}
	wg                  sync.WaitGroup
	maintenanceInterval time.Duration
	lastMaintenance     time.Time
	isRunning           bool
	mutex               sync.RWMutex
}

// NewServer creates a new WebSocket server
func NewServer(
	logger logger.Service,
	connectionService connection.Service,
) *Server {
	return &Server{
		logger:              logger,
		connectionService:   connectionService,
		shutdown:            make(chan struct{}),
		maintenanceInterval: 5 * time.Minute,
		lastMaintenance:     clock.Now(),
	}
}

// Run starts the WebSocket server's background operations
func (s *Server) Run() {
	s.mutex.Lock()
	if s.isRunning {
		s.mutex.Unlock()
		return
	}
	s.isRunning = true
	s.mutex.Unlock()

	s.logger.Info(log.WebSocket, "WebSocket server started")

	// Start maintenance tasks
	s.wg.Add(1)
	go s.maintenanceLoop()
}

// maintenanceLoop runs periodic maintenance tasks
func (s *Server) maintenanceLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.maintenanceInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.performMaintenance()
		case <-s.shutdown:
			s.logger.Info(log.WebSocket, "Maintenance loop shutting down")
			return
		}
	}
}

// performMaintenance performs periodic maintenance tasks
func (s *Server) performMaintenance() {
	now := clock.Now()
	elapsed := now.Sub(s.lastMaintenance)

	s.logger.Info(log.WebSocket, `Performing WebSocket server maintenance lastMaintenance=%v elapsed=%v`, s.lastMaintenance.Format(time.RFC3339),
		elapsed.String())

	// Clean stale connections if the connection service is available
	if s.connectionService != nil {
		count, err := s.connectionService.CleanStaleConnections()
		if err != nil {
			s.logger.Error(log.WebSocket, `Error cleaning stale connections error=%v`, err)
		} else {
			s.logger.Info(log.WebSocket, `Cleaned stale connections count=%v`, count)
		}
	}

	s.lastMaintenance = now
}

// Close gracefully shuts down the WebSocket server
func (s *Server) Close() {
	s.mutex.Lock()
	if !s.isRunning {
		s.mutex.Unlock()
		return
	}
	s.isRunning = false
	s.mutex.Unlock()

	s.logger.Info(log.WebSocket, "WebSocket server shutting down")

	// Signal all background goroutines to stop
	close(s.shutdown)

	// Wait for all goroutines to finish with a timeout
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	// Wait with timeout
	select {
	case <-done:
		s.logger.Info(log.WebSocket, "WebSocket server shutdown completed successfully")
	case <-time.After(10 * time.Second):
		s.logger.Warn(log.WebSocket, "WebSocket server shutdown timed out")
	}
}

// SetMaintenanceInterval changes the maintenance interval
func (s *Server) SetMaintenanceInterval(interval time.Duration) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.maintenanceInterval = interval
}

// IsRunning returns whether the server is currently running
func (s *Server) IsRunning() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.isRunning
}

// RunMaintenance triggers maintenance immediately
func (s *Server) RunMaintenance() {
	s.performMaintenance()
}
