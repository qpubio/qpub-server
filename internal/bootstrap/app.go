package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"sync"

	"github.com/qpubio/qpub-server/internal/api/websocket"
	"github.com/qpubio/qpub-server/internal/bootstrap/container"
	"github.com/qpubio/qpub-server/internal/config"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/infrastructure/nats"
	"github.com/qpubio/qpub-server/internal/infrastructure/redis"
	"github.com/qpubio/qpub-server/internal/shared/apikey"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"github.com/qpubio/qpub-server/internal/shared/permission"
	"github.com/qpubio/qpub-server/internal/shared/token"
	"github.com/qpubio/qpub-server/internal/shared/type/log"

	"gorm.io/gorm"
)

// App holds all dependencies for the data-plane server.
type App struct {
	ctx    context.Context
	cancel context.CancelFunc

	config       *config.Config
	logger       logger.Service
	instanceID   id.ULID
	apikeyParser *apikey.Parser
	token        *token.Service
	permission   permission.Service

	db    *gorm.DB
	redis redis.Service
	nats  nats.Service

	container *container.Container
	handlers  *HandlerContainer

	httpServers map[string]*http.Server
	wsServer    *websocket.Server

	wg             sync.WaitGroup
	cleanup        *Cleanup
	postLoggerInit []func()
}

type Cleanup struct {
	handlers []func() error
}

func NewCleanup() *Cleanup {
	return &Cleanup{handlers: make([]func() error, 0)}
}

func (c *Cleanup) Register(fn func() error) {
	c.handlers = append(c.handlers, fn)
}

func (c *Cleanup) Run() error {
	for i := len(c.handlers) - 1; i >= 0; i-- {
		if err := c.handlers[i](); err != nil {
			return err
		}
	}
	return nil
}

func New() (*App, error) {
	ctx, cancel := context.WithCancel(context.Background())
	return &App{
		ctx:         ctx,
		cancel:      cancel,
		httpServers: make(map[string]*http.Server),
		cleanup:     NewCleanup(),
	}, nil
}

func (a *App) Start() error {
	if err := a.setupBrand(); err != nil {
		return err
	}
	if err := a.setupConfig(); err != nil {
		return err
	}
	if err := a.setupLogger(); err != nil {
		return err
	}
	if err := a.setupTimezone(); err != nil {
		return fmt.Errorf("failed to setup timezone: %w", err)
	}
	if err := a.setupID(); err != nil {
		return err
	}
	if err := a.setupInstanceID(); err != nil {
		return fmt.Errorf("failed to setup instance ID: %w", err)
	}
	if err := setupRedis(a); err != nil {
		return fmt.Errorf("failed to setup redis: %w", err)
	}
	if err := a.setupAPIKeyParser(); err != nil {
		return err
	}
	if err := a.setupToken(); err != nil {
		return err
	}
	if err := a.setupPermission(); err != nil {
		return err
	}
	if err := setupDatabase(a); err != nil {
		return fmt.Errorf("failed to setup database: %w", err)
	}
	if err := setupNATS(a); err != nil {
		return fmt.Errorf("failed to setup nats: %w", err)
	}
	if err := a.setupContainer(); err != nil {
		return fmt.Errorf("failed to setup dependency container: %w", err)
	}
	if err := a.registerServiceModules(); err != nil {
		return fmt.Errorf("failed to register service modules: %w", err)
	}
	if err := a.initializeLifecycleServices(); err != nil {
		return fmt.Errorf("failed to initialize lifecycle services: %w", err)
	}
	if err := a.setupHandlers(); err != nil {
		return err
	}
	if err := a.setupHTTPServers(); err != nil {
		return err
	}
	if err := a.setupWebSocketServer(); err != nil {
		return err
	}
	return nil
}

func (a *App) Shutdown() error {
	a.cancel()
	if err := a.cleanup.Run(); err != nil {
		a.logger.Error(log.App, "Cleanup error: %v", err)
	}
	a.wg.Wait()
	return nil
}

func (a *App) DB() *gorm.DB                    { return a.db }
func (a *App) Redis() redis.Service            { return a.redis }
func (a *App) NATS() nats.Service              { return a.nats }
func (a *App) Logger() logger.Service          { return a.logger }
func (a *App) Config() *config.Config          { return a.config }
func (a *App) Context() context.Context        { return a.ctx }
func (a *App) Container() *container.Container { return a.container }
func (a *App) InstanceID() id.ULID             { return a.instanceID }

func (a *App) GetService(serviceType reflect.Type) (interface{}, error) {
	return a.container.Get(serviceType)
}

func (a *App) GetServiceByPtr(ptr interface{}) (interface{}, error) {
	return a.container.Get(reflect.TypeOf(ptr).Elem())
}
