package container

// Module is an interface for registering a group of related services
type Module interface {
	Name() string
	Register(c *Container) error
	Order() int
}

type ModuleBase struct {
	name  string
	order int
}

func NewModule(name string, order int) ModuleBase {
	return ModuleBase{name: name, order: order}
}

func (m ModuleBase) Name() string { return m.name }
func (m ModuleBase) Order() int   { return m.order }

func RegisterModules(c *Container, modules ...Module) error {
	sorted := make([]Module, len(modules))
	copy(sorted, modules)
	for i := 1; i < len(sorted); i++ {
		j := i
		for j > 0 && sorted[j-1].Order() > sorted[j].Order() {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
			j--
		}
	}
	for _, module := range sorted {
		if err := module.Register(c); err != nil {
			return err
		}
	}
	return nil
}
