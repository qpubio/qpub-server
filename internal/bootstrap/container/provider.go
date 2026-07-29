package container

import "reflect"

// Provider is an interface for registering service descriptors
type Provider interface {
	Register(c *Container) error
}

func (c *Container) RegisterInterface(interfaceType reflect.Type, factory factoryFunc) {
	c.Register(interfaceType, factory)
}

func (c *Container) RegisterType(concreteType reflect.Type, factory factoryFunc) {
	c.Register(concreteType, factory)
}

func (c *Container) RegisterDescriptor(descriptor *ServiceDescriptor) {
	c.Register(descriptor.Interface, descriptor.Factory)
}

func (c *Container) RegisterProvider(provider Provider) error {
	return provider.Register(c)
}

func TypeOf[T any]() reflect.Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}

func InterfaceOf[T any]() reflect.Type {
	var t T
	return reflect.TypeOf(&t).Elem()
}
