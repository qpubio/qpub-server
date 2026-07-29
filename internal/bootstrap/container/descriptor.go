package container

import "reflect"

type ServiceDescriptor struct {
	Interface      reflect.Type
	Implementation reflect.Type
	Dependencies   []reflect.Type
	Factory        factoryFunc
	Singleton      bool
	EagerLoad      bool
	HasInitialize  bool
	HasShutdown    bool
}

func NewDescriptor(interfaceType reflect.Type, factory factoryFunc) *ServiceDescriptor {
	return &ServiceDescriptor{
		Interface: interfaceType,
		Factory:   factory,
		Singleton: true,
	}
}

type Lifecycle interface {
	Initialize() error
	Shutdown() error
}
