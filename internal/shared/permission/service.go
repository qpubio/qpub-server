package permission

// Service provides permission checking functionality
type Service interface {
	CanSubscribe(data []byte, resource string) (bool, error)
	CanPublish(data []byte, resource string) (bool, error)
	CanStats(data []byte, resource string) (bool, error)
	CanLogs(data []byte, resource string) (bool, error)
	CanEnqueue(data []byte, resource string) (bool, error)
	CanDequeue(data []byte, resource string) (bool, error)
}

type service struct{}

// NewService creates a new permission service
func NewService() Service {
	return &service{}
}

func (s *service) CanSubscribe(data []byte, resource string) (bool, error) {
	// Platform channels use dedicated actions, not general subscribe.
	switch resource {
	case ChannelStats:
		return s.CanStats(data, resource)
	case ChannelLogs:
		return s.CanLogs(data, resource)
	}

	p, err := FromJSON(data)
	if err != nil {
		return false, err
	}
	return p.Can(resource, ActionSubscribe), nil
}

func (s *service) CanPublish(data []byte, resource string) (bool, error) {
	p, err := FromJSON(data)
	if err != nil {
		return false, err
	}
	return p.Can(resource, ActionPublish), nil
}

func (s *service) CanStats(data []byte, resource string) (bool, error) {
	p, err := FromJSON(data)
	if err != nil {
		return false, err
	}
	return p.Can(resource, ActionStats), nil
}

func (s *service) CanLogs(data []byte, resource string) (bool, error) {
	p, err := FromJSON(data)
	if err != nil {
		return false, err
	}
	return p.Can(resource, ActionLogs), nil
}

func (s *service) CanEnqueue(data []byte, resource string) (bool, error) {
	p, err := FromJSON(data)
	if err != nil {
		return false, err
	}
	if p.Can(resource, ActionEnqueue) {
		return true, nil
	}
	return p.Can(resource, ActionPublish), nil
}

func (s *service) CanDequeue(data []byte, resource string) (bool, error) {
	p, err := FromJSON(data)
	if err != nil {
		return false, err
	}
	if p.Can(resource, ActionDequeue) {
		return true, nil
	}
	return p.Can(resource, ActionSubscribe), nil
}
