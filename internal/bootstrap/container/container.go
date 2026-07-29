package container

import (
	"fmt"
	"reflect"
)

// Container is a dependency injection container that manages service lifecycles
type Container struct {
	services          map[reflect.Type]interface{}
	factories         map[reflect.Type]factoryFunc
	initializing      map[reflect.Type]bool
	lifecycleServices []interface{}
}

type factoryFunc func(c *Container) (interface{}, error)

func New() *Container {
	return &Container{
		services:          make(map[reflect.Type]interface{}),
		factories:         make(map[reflect.Type]factoryFunc),
		initializing:      make(map[reflect.Type]bool),
		lifecycleServices: []interface{}{},
	}
}

func (c *Container) Register(serviceType reflect.Type, factory factoryFunc) {
	c.factories[serviceType] = factory
}

func (c *Container) RegisterInstance(serviceType reflect.Type, instance interface{}) {
	c.services[serviceType] = instance
}

func (c *Container) Get(serviceType reflect.Type) (interface{}, error) {
	if service, exists := c.services[serviceType]; exists {
		return service, nil
	}
	if c.initializing[serviceType] {
		return nil, fmt.Errorf("circular dependency detected while resolving type %v", serviceType)
	}
	factory, exists := c.factories[serviceType]
	if !exists {
		return nil, fmt.Errorf("no factory registered for type %v", serviceType)
	}
	c.initializing[serviceType] = true
	defer delete(c.initializing, serviceType)

	service, err := factory(c)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize service %v: %w", serviceType, err)
	}
	if lifecycle, ok := service.(Lifecycle); ok {
		if err := lifecycle.Initialize(); err != nil {
			return nil, fmt.Errorf("failed to initialize lifecycle service %v: %w", serviceType, err)
		}
		c.lifecycleServices = append(c.lifecycleServices, service)
	}
	c.services[serviceType] = service
	return service, nil
}

func GetTyped[T any](c *Container) (T, error) {
	var zero T
	serviceType := reflect.TypeOf((*T)(nil)).Elem()
	service, err := c.Get(serviceType)
	if err != nil {
		return zero, err
	}
	typedService, ok := service.(T)
	if !ok {
		return zero, fmt.Errorf("service of type %v is not assignable to %T", serviceType, zero)
	}
	return typedService, nil
}

func (c *Container) Shutdown() error {
	var lastErr error
	for i := len(c.lifecycleServices) - 1; i >= 0; i-- {
		if lifecycle, ok := c.lifecycleServices[i].(Lifecycle); ok {
			if err := lifecycle.Shutdown(); err != nil {
				lastErr = err
			}
		}
	}
	return lastErr
}
