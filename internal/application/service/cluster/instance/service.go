package instance

import (
	"fmt"
	"time"

	"github.com/qpubio/qpub-server/internal/config"
	"github.com/qpubio/qpub-server/internal/domain/cluster/instance"
	"github.com/qpubio/qpub-server/internal/domain/project/stat/realtime"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/clock"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"github.com/qpubio/qpub-server/internal/shared/ptr"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
)

type Service struct {
	config      *config.Config
	logger      logger.Service
	instanceID  id.ULID
	repository  instance.Repository
	statService realtime.Service
}

func NewService(
	config *config.Config,
	logger logger.Service,
	instanceID id.ULID,
	repository instance.Repository,
	statService realtime.Service,
) instance.Service {
	return &Service{
		config:      config,
		logger:      logger,
		instanceID:  instanceID,
		repository:  repository,
		statService: statService,
	}
}

func (s *Service) Register() error {
	// Check if instance already exists
	existing, _ := s.Get()

	if existing != (instance.ServerInstance{}) {
		// Update the existing instance
		err := existing.Update(instance.UpdateParams{
			ClusterScope:  s.config.Infrastructure.Cluster.ClusterScope,
			ClusterEnv:    s.config.Infrastructure.Cluster.ClusterEnv,
			ClusterRegion: s.config.Infrastructure.Cluster.ClusterRegion,
			ClusterZone:   s.config.Infrastructure.Cluster.ClusterZone,
			VMScope:       s.config.Infrastructure.Cluster.VMScope,
			VMEnv:         s.config.Infrastructure.Cluster.VMEnv,
			VMRegion:      s.config.Infrastructure.Cluster.VMRegion,
			VMZone:        s.config.Infrastructure.Cluster.VMZone,
			VMRole:        s.config.Infrastructure.Cluster.VMRole,
			VMSeq:         s.config.Infrastructure.Cluster.VMSeq,
			Status:        instance.StatusActive,
		})
		if err != nil {
			return err
		}

		if err := s.repository.Update(&existing); err != nil {
			return fmt.Errorf("error updating existing instance: %w", err)
		}

		s.logger.Info(log.Instance, `Updated existing instance instanceID=%v vmName=%v clusterName=%v`, s.instanceID,
			s.config.Infrastructure.Cluster.VMName,
			s.config.Infrastructure.Cluster.Name)
		return nil
	}

	// Create a new instance
	inst, err := instance.Create(instance.CreateParams{
		InstanceID:    s.instanceID,
		ClusterScope:  s.config.Infrastructure.Cluster.ClusterScope,
		ClusterEnv:    s.config.Infrastructure.Cluster.ClusterEnv,
		ClusterRegion: s.config.Infrastructure.Cluster.ClusterRegion,
		ClusterZone:   s.config.Infrastructure.Cluster.ClusterZone,
		VMScope:       s.config.Infrastructure.Cluster.VMScope,
		VMEnv:         s.config.Infrastructure.Cluster.VMEnv,
		VMRegion:      s.config.Infrastructure.Cluster.VMRegion,
		VMZone:        s.config.Infrastructure.Cluster.VMZone,
		VMRole:        s.config.Infrastructure.Cluster.VMRole,
		VMSeq:         s.config.Infrastructure.Cluster.VMSeq,
	})
	if err != nil {
		return err
	}

	s.logger.Debug(log.Instance, `Creating new instance instanceData=%v`, inst)
	if _, err := s.repository.Create(inst); err != nil {
		return fmt.Errorf("error creating new instance: %w", err)
	}

	s.logger.Info(log.Instance, `Registered new instance instanceID=%v vmName=%v clusterName=%v`, s.instanceID,
		s.config.Infrastructure.Cluster.VMName,
		s.config.Infrastructure.Cluster.Name)
	return nil
}

func (s *Service) Heartbeat() error {
	inst, err := s.Get()
	if err != nil {
		return fmt.Errorf("error getting instance by instance ID: %w", err)
	}
	err = inst.Update(instance.UpdateParams{
		Status: instance.StatusActive,
	})
	if err != nil {
		return err
	}

	err = s.repository.Update(&inst)
	if err == nil {
		s.logger.Info(log.Instance, `Instance heartbeat instanceID=%v vmName=%v`, s.instanceID,
			s.config.Infrastructure.Cluster.VMName)
	}

	return err
}

func (s *Service) Deregister() error {
	// Find the instance first
	inst, err := s.Get()
	if err != nil {
		return fmt.Errorf("error finding instance to deregister: %w", err)
	}

	// Mark it as inactive
	err = inst.Update(instance.UpdateParams{
		Status: instance.StatusInactive,
	})
	if err != nil {
		return err
	}

	err = s.repository.Update(&inst)
	if err != nil {
		return fmt.Errorf("error updating instance: %w", err)
	}

	// Clean up resources associated with this instance
	if err := s.cleanupInstanceResources(); err != nil {
		s.logger.Warn(log.Instance, `Error cleaning up instance resources instanceID=%v error=%v`, s.instanceID,
			err)
		// Continue even if cleanup failed
	}

	s.logger.Info(log.Instance, `Deregistered instance instanceID=%v`, s.instanceID)
	return nil
}

func (s *Service) CleanupStaleInstances() (int, error) {
	threshold := clock.Now().Add(-time.Duration(s.config.Infrastructure.Instance.InactivityTimeout) * time.Second)

	// Find all stale instances
	staleInstances, err := s.repository.ListStale(threshold)
	if err != nil {
		return 0, fmt.Errorf("error listing stale instances: %w", err)
	}

	if len(staleInstances) == 0 {
		return 0, nil
	}

	s.logger.Info(log.Instance, `Found stale instances count=%v threshold=%v`, len(staleInstances),
		threshold)

	deregisteredCount := 0

	// Deregister each stale instance
	for _, inst := range staleInstances {
		// Mark the instance as inactive
		err = inst.Update(instance.UpdateParams{
			Status: instance.StatusInactive,
		})
		if err != nil {
			return 0, err
		}

		if err := s.repository.Update(inst); err != nil {
			s.logger.Warn(log.Instance, `Error marking instance as inactive instanceID=%v error=%v`, inst.InstanceID,
				err)
			continue
		}

		// Clean up resources associated with this instance
		if err := s.cleanupInstanceResources(); err != nil {
			s.logger.Warn(log.Instance, `Error cleaning up instance resources instanceID=%v error=%v`, inst.InstanceID,
				err)
			// Continue anyway
		}

		deregisteredCount++

		s.logger.Info(log.Instance, `Deregistered stale instance instanceID=%v lastHeartbeat=%v`, inst.InstanceID,
			inst.UpdatedAt)
	}

	return deregisteredCount, nil
}

func (s *Service) Get() (instance.ServerInstance, error) {
	instPtr, err := s.repository.FindByInstanceID(s.instanceID)
	if err != nil {
		return instance.ServerInstance{}, fmt.Errorf("error getting instance by instance ID: %w", err)
	}
	inst := ptr.ToValue(instPtr)
	return inst, nil
}

// Helper method to clean up resources associated with an instance
func (s *Service) cleanupInstanceResources() error {
	// Define stats to reset
	statsToReset := []realtime.KeyType{
		realtime.KeyConnection,
		realtime.KeyChannel,
		realtime.KeySubscriber,
	}

	// Reset each statistic for this instance
	for _, keyType := range statsToReset {
		// Generate pattern for this instance and stat type
		pattern := realtime.NewKey(keyType, s.instanceID, 0).Pattern()

		s.logger.Info(log.Instance, `Resetting stats for deregistered instance instanceID=%v statType=%v pattern=%v`, s.instanceID,
			keyType,
			pattern)

		if err := s.statService.ResetByPattern(pattern); err != nil {
			s.logger.Warn(log.Instance, `Error resetting stats for instance instanceID=%v statType=%v error=%v`, s.instanceID,
				keyType,
				err)
			// Continue with other stat types
		}
	}

	return nil
}
