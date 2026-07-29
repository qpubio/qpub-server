package broker

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	domainBroker "github.com/qpubio/qpub-server/internal/domain/queue/broker"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/infrastructure/nats"
	"github.com/qpubio/qpub-server/internal/shared/type/log"

	natsgo "github.com/nats-io/nats.go"
)

type repository struct {
	nats   nats.Service
	logger logger.Service
	js     natsgo.JetStreamContext
	mu     sync.RWMutex
	streams map[string]bool
}

func NewRepository(natsService nats.Service, logger logger.Service) (domainBroker.Repository, error) {
	js, err := natsService.JetStream()
	if err != nil {
		return nil, fmt.Errorf("jetstream unavailable: %w", err)
	}

	return &repository{
		nats:    natsService,
		logger:  logger,
		js:      js,
		streams: make(map[string]bool),
	}, nil
}

func (r *repository) EnsureStream(subject string, retention time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.streams[subject] {
		return nil
	}

	streamName := streamNameForSubject(subject)
	_, err := r.js.StreamInfo(streamName)
	if err == nil {
		r.streams[subject] = true
		return nil
	}

	maxAge := retention
	if maxAge <= 0 {
		maxAge = 7 * 24 * time.Hour
	}

	_, err = r.js.AddStream(&natsgo.StreamConfig{
		Name:      streamName,
		Subjects:  []string{subjectWild(subject)},
		Retention: natsgo.WorkQueuePolicy,
		MaxAge:    maxAge,
		Storage:   natsgo.FileStorage,
	})
	if err != nil {
		return fmt.Errorf("failed to create stream %s: %w", streamName, err)
	}

	r.streams[subject] = true
	r.logger.Info(log.Queue, "Created JetStream stream stream=%s subject=%s", streamName, subject)
	return nil
}

func (r *repository) Publish(ctx context.Context, subject string, data []byte) (uint64, error) {
	if err := r.EnsureStream(subject, 7*24*time.Hour); err != nil {
		return 0, err
	}

	ack, err := r.js.Publish(subject, data, natsgo.Context(ctx))
	if err != nil {
		return 0, fmt.Errorf("failed to publish job: %w", err)
	}
	return ack.Sequence, nil
}

func (r *repository) Pull(ctx context.Context, subject string, batchSize int, wait time.Duration) ([]domainBroker.Message, error) {
	if err := r.EnsureStream(subject, 7*24*time.Hour); err != nil {
		return nil, err
	}

	if batchSize <= 0 {
		batchSize = 1
	}

	consumerName := consumerNameForSubject(subject)
	_, err := r.js.ConsumerInfo(streamNameForSubject(subject), consumerName)
	if err != nil {
		_, err = r.js.AddConsumer(streamNameForSubject(subject), &natsgo.ConsumerConfig{
			Durable:       consumerName,
			FilterSubject: subject,
			AckPolicy:     natsgo.AckExplicitPolicy,
			MaxAckPending: 1000,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create consumer: %w", err)
		}
	}

	sub, err := r.js.PullSubscribe(subject, consumerName, natsgo.Bind(streamNameForSubject(subject), consumerName))
	if err != nil {
		return nil, fmt.Errorf("failed to pull subscribe: %w", err)
	}
	defer sub.Unsubscribe()

	msgs, err := sub.Fetch(batchSize, natsgo.MaxWait(wait))
	if err != nil {
		if err == natsgo.ErrTimeout {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}

	result := make([]domainBroker.Message, 0, len(msgs))
	for _, msg := range msgs {
		meta, _ := msg.Metadata()
		seq := uint64(0)
		if meta != nil {
			seq = meta.Sequence.Stream
		}
		result = append(result, domainBroker.Message{
			Subject:  msg.Subject,
			Data:     append([]byte(nil), msg.Data...),
			Sequence: seq,
		})
	}

	return result, nil
}

func (r *repository) Ack(ctx context.Context, subject string, sequence uint64) error {
	consumerName := consumerNameForSubject(subject)
	sub, err := r.js.PullSubscribe(subject, consumerName, natsgo.Bind(streamNameForSubject(subject), consumerName))
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()

	return nil
}

func (r *repository) Nack(ctx context.Context, subject string, sequence uint64, delay time.Duration) error {
	return nil
}

func (r *repository) PublishDLQ(ctx context.Context, subject string, data []byte) error {
	return r.EnsureStream(subject, 30*24*time.Hour)
}

func (r *repository) Shutdown(ctx context.Context) error {
	return nil
}

func streamNameForSubject(subject string) string {
	safe := strings.ReplaceAll(subject, ".", "_")
	safe = strings.ReplaceAll(safe, "*", "all")
	if len(safe) > 64 {
		safe = safe[:64]
	}
	return "QPUB_" + safe
}

func subjectWild(subject string) string {
	if strings.Contains(subject, "*") {
		return subject
	}
	return subject
}

func consumerNameForSubject(subject string) string {
	name := strings.ReplaceAll(subject, ".", "_")
	if len(name) > 64 {
		name = name[len(name)-64:]
	}
	return "cons_" + name
}
