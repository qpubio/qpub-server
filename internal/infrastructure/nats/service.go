package nats

import (
	"fmt"
	infrastructure "github.com/qpubio/qpub-server/internal/config/infrastructure"

	"github.com/nats-io/nats.go"
)

type Service interface {
	Connect() error
	Close() error
	Publish(subject string, data []byte) error
	Subscribe(subject string, handler nats.MsgHandler) (*nats.Subscription, error)
	QueueSubscribe(subject, queue string, handler nats.MsgHandler) (*nats.Subscription, error)
	Conn() *nats.Conn
	JetStream() (nats.JetStreamContext, error)
}

type service struct {
	conn   *nats.Conn
	config *infrastructure.NATS
}

func New(config *infrastructure.NATS) (Service, error) {
	if config == nil {
		return nil, fmt.Errorf("nats config cannot be nil")
	}

	return &service{
		config: config,
	}, nil
}

func (s *service) Connect() error {
	opts := []nats.Option{
		nats.ReconnectWait(s.config.Reconnect.Wait),
		nats.MaxReconnects(s.config.Reconnect.MaxAttempts),
		nats.ReconnectBufSize(s.config.Reconnect.Buffer),
		nats.Timeout(s.config.Timeout.Connect),
		nats.PingInterval(s.config.Timeout.Ping),
	}

	// Add authentication if configured
	if s.config.Username != "" && s.config.Password != "" {
		opts = append(opts, nats.UserInfo(s.config.Username, s.config.Password))
	}

	// Add TLS if configured
	if s.config.TLS.Enable {
		opts = append(opts, nats.ClientCert(
			s.config.TLS.Cert,
			s.config.TLS.Key,
		))
		if s.config.TLS.CA != "" {
			opts = append(opts, nats.RootCAs(s.config.TLS.CA))
		}
	}

	// Connect to NATS
	nc, err := nats.Connect(s.config.URL, opts...)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	s.conn = nc
	return nil
}

func (s *service) Close() error {
	if s.conn != nil {
		s.conn.Close()
	}
	return nil
}

func (s *service) Publish(subject string, data []byte) error {
	if s.conn == nil {
		return fmt.Errorf("NATS connection not established")
	}
	return s.conn.Publish(subject, data)
}

func (s *service) Subscribe(subject string, handler nats.MsgHandler) (*nats.Subscription, error) {
	if s.conn == nil {
		return nil, fmt.Errorf("NATS connection not established")
	}
	return s.conn.Subscribe(subject, handler)
}

func (s *service) QueueSubscribe(subject, queue string, handler nats.MsgHandler) (*nats.Subscription, error) {
	if s.conn == nil {
		return nil, fmt.Errorf("NATS connection not established")
	}
	return s.conn.QueueSubscribe(subject, queue, handler)
}

func (s *service) Conn() *nats.Conn {
	return s.conn
}

func (s *service) JetStream() (nats.JetStreamContext, error) {
	if s.conn == nil {
		return nil, fmt.Errorf("NATS connection not established")
	}
	return s.conn.JetStream()
}
