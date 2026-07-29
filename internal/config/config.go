package config

import (
	"fmt"

	"github.com/qpubio/qpub-server/internal/config/app"
	"github.com/qpubio/qpub-server/internal/config/auth"
	"github.com/qpubio/qpub-server/internal/config/infrastructure"
	"github.com/qpubio/qpub-server/internal/config/shared"
)

// Config is the root configuration for the data-plane server.
type Config struct {
	App app.App

	Auth struct {
		Token auth.Token
	}

	Infrastructure struct {
		Logger   infrastructure.Logger
		Database infrastructure.Database
		Redis    infrastructure.Redis
		NATS     infrastructure.NATS
		Queue    infrastructure.Queue
		Cluster  infrastructure.Cluster
		Instance infrastructure.Instance
		Server   infrastructure.Server
		CORS     infrastructure.CORS
		Time     infrastructure.Time
	}

	Shared struct {
		ID shared.ID
	}
}

func NewConfig() (*Config, error) {
	cfg := &Config{}

	cfgApp, err := app.NewApp()
	if err != nil {
		return nil, fmt.Errorf("failed to parse app config: %w", err)
	}
	cfg.App = *cfgApp

	cfgToken, err := auth.NewToken()
	if err != nil {
		return nil, fmt.Errorf("failed to parse auth token config: %w", err)
	}
	cfg.Auth.Token = *cfgToken

	cfgLogger, err := infrastructure.NewLogger()
	if err != nil {
		return nil, fmt.Errorf("failed to parse logger config: %w", err)
	}
	cfg.Infrastructure.Logger = *cfgLogger

	cfgDatabase, err := infrastructure.NewDatabase()
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}
	cfg.Infrastructure.Database = *cfgDatabase

	cfgRedis, err := infrastructure.NewRedis()
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis config: %w", err)
	}
	cfg.Infrastructure.Redis = *cfgRedis

	cfgNATS, err := infrastructure.NewNATS()
	if err != nil {
		return nil, fmt.Errorf("failed to parse nats config: %w", err)
	}
	cfg.Infrastructure.NATS = *cfgNATS

	cfgQueue, err := infrastructure.NewQueue()
	if err != nil {
		return nil, fmt.Errorf("failed to parse queue config: %w", err)
	}
	cfg.Infrastructure.Queue = *cfgQueue

	cfgCluster, err := infrastructure.NewCluster()
	if err != nil {
		return nil, fmt.Errorf("failed to parse cluster config: %w", err)
	}
	cfg.Infrastructure.Cluster = *cfgCluster

	cfgInstance, err := infrastructure.NewInstance()
	if err != nil {
		return nil, fmt.Errorf("failed to parse instance config: %w", err)
	}
	cfg.Infrastructure.Instance = *cfgInstance

	cfgServer, err := infrastructure.NewServer()
	if err != nil {
		return nil, fmt.Errorf("failed to parse server config: %w", err)
	}
	cfg.Infrastructure.Server = *cfgServer

	cfgCORS, err := infrastructure.NewCORS()
	if err != nil {
		return nil, fmt.Errorf("failed to parse cors config: %w", err)
	}
	cfg.Infrastructure.CORS = *cfgCORS

	cfgTime, err := infrastructure.NewTime()
	if err != nil {
		return nil, fmt.Errorf("failed to parse time config: %w", err)
	}
	cfg.Infrastructure.Time = *cfgTime

	cfgID, err := shared.NewID()
	if err != nil {
		return nil, fmt.Errorf("failed to parse shared id config: %w", err)
	}
	cfg.Shared.ID = *cfgID

	return cfg, nil
}
