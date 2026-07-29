package egress

import (
	"sync"
	"time"

	"github.com/qpubio/qpub-server/internal/domain/messaging/backpressure"
	"github.com/qpubio/qpub-server/internal/domain/messaging/connection"
	domainTelemetry "github.com/qpubio/qpub-server/internal/domain/messaging/telemetry"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
)

const (
	defaultQueueCapacity        = 256
	defaultBlockTimeout         = 50 * time.Millisecond
	defaultSlowConsumerThreshold = 10
)

// PipelineParams configures a per-connection outbound pipeline.
type PipelineParams struct {
	ConnID                   id.ULID
	ProjectID                id.Int
	InstanceID               id.ULID
	ConnectionService        connection.Service
	TelemetryService         domainTelemetry.Service
	Logger                   logger.Service
	Capacity                 int
	BlockTimeout             time.Duration
	SlowConsumerThreshold    int
	OnSlowConsumer           func(connID id.ULID)
}

// Pipeline is a single-writer outbound queue for one connection.
type Pipeline struct {
	connID                id.ULID
	projectID             id.Int
	instanceID            id.ULID
	connectionService     connection.Service
	telemetryService      domainTelemetry.Service
	logger                logger.Service
	queue                 chan []byte
	stop                  chan struct{}
	done                  chan struct{}
	blockTimeout          time.Duration
	slowConsumerThreshold int
	consecutiveDrops      int
	onSlowConsumer        func(connID id.ULID)
}

// NewPipeline creates an outbound pipeline. Call Start before Enqueue.
func NewPipeline(params PipelineParams) *Pipeline {
	capacity := params.Capacity
	if capacity < 1 {
		capacity = defaultQueueCapacity
	}
	blockTimeout := params.BlockTimeout
	if blockTimeout <= 0 {
		blockTimeout = defaultBlockTimeout
	}
	threshold := params.SlowConsumerThreshold
	if threshold < 1 {
		threshold = defaultSlowConsumerThreshold
	}

	return &Pipeline{
		connID:                params.ConnID,
		projectID:             params.ProjectID,
		instanceID:            params.InstanceID,
		connectionService:     params.ConnectionService,
		telemetryService:      params.TelemetryService,
		logger:                params.Logger,
		queue:                 make(chan []byte, capacity),
		stop:                  make(chan struct{}),
		done:                  make(chan struct{}),
		blockTimeout:          blockTimeout,
		slowConsumerThreshold: threshold,
		onSlowConsumer:        params.OnSlowConsumer,
	}
}

// Start launches the writer goroutine.
func (p *Pipeline) Start() {
	go p.run()
}

// Stop closes the pipeline and waits for the writer to exit.
func (p *Pipeline) Stop() {
	close(p.stop)
	<-p.done
}

// Enqueue adds a payload using block-with-timeout backpressure at connection outbound.
func (p *Pipeline) Enqueue(payload []byte) error {
	if len(payload) == 0 {
		return nil
	}

	msg := append([]byte(nil), payload...)

	select {
	case <-p.stop:
		return connection.ErrConnectionClosed
	default:
	}

	timer := time.NewTimer(p.blockTimeout)
	defer timer.Stop()

	select {
	case <-p.stop:
		return connection.ErrConnectionClosed
	case p.queue <- msg:
		return nil
	case <-timer.C:
		p.handleDrop(backpressure.DropReasonWriteTimeout, len(p.queue))
		return nil
	}
}

func (p *Pipeline) run() {
	defer close(p.done)

	for {
		select {
		case <-p.stop:
			return
		case msg := <-p.queue:
			p.deliver(msg)
		}
	}
}

func (p *Pipeline) deliver(payload []byte) {
	if err := p.connectionService.Send(p.connID, payload); err != nil {
		p.logger.Error(log.MessagingConnection,
			"Outbound pipeline failed to deliver message connectionID=%s projectID=%d error=%v",
			p.connID, p.projectID, err)

		evt := domainTelemetry.NewEvent(
			domainTelemetry.TypeOutboundFailed,
			p.projectID,
			p.instanceID,
			domainTelemetry.OutboundFailedData{
				ConnectionID: p.connID,
				Reason:       err.Error(),
			},
		)
		if recordErr := p.telemetryService.Record(evt); recordErr != nil {
			p.logger.Warn(log.MessagingConnection,
				"Failed to record outbound failure telemetry connectionID=%s error=%v",
				p.connID, recordErr)
		}
		return
	}

	p.consecutiveDrops = 0

	evt := domainTelemetry.NewEvent(
		domainTelemetry.TypeOutboundDelivered,
		p.projectID,
		p.instanceID,
		domainTelemetry.OutboundDeliveredData{
			ConnectionID: p.connID,
			ByteSize:     int64(len(payload)),
		},
	)
	if err := p.telemetryService.Record(evt); err != nil {
		p.logger.Warn(log.MessagingConnection,
			"Failed to record outbound delivery telemetry connectionID=%s error=%v",
			p.connID, err)
	}
}

func (p *Pipeline) handleDrop(reason backpressure.DropReason, queueDepth int) {
	p.consecutiveDrops++
	p.recordDropped(reason, queueDepth)

	if p.consecutiveDrops >= p.slowConsumerThreshold && p.onSlowConsumer != nil {
		p.onSlowConsumer(p.connID)
	}
}

func (p *Pipeline) recordDropped(reason backpressure.DropReason, queueDepth int) {
	p.logger.Warn(log.MessagingConnection,
		"Outbound message dropped connectionID=%s projectID=%d queueDepth=%d reason=%s",
		p.connID, p.projectID, queueDepth, reason)

	evt := domainTelemetry.NewEvent(
		domainTelemetry.TypeOutboundDropped,
		p.projectID,
		p.instanceID,
		domainTelemetry.OutboundDroppedData{
			ConnectionID: p.connID,
			Reason:       reason,
			QueueDepth:   queueDepth,
		},
	)
	if err := p.telemetryService.Record(evt); err != nil {
		p.logger.Warn(log.MessagingConnection,
			"Failed to record outbound drop telemetry connectionID=%s error=%v",
			p.connID, err)
	}
}

// Registry tracks outbound pipelines keyed by connection ID.
type Registry struct {
	mu        sync.RWMutex
	pipelines map[id.ULID]*Pipeline
}

// NewRegistry creates an empty pipeline registry.
func NewRegistry() *Registry {
	return &Registry{
		pipelines: make(map[id.ULID]*Pipeline),
	}
}

// Register stores and starts a pipeline for a connection.
func (r *Registry) Register(connID id.ULID, pipeline *Pipeline) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pipelines[connID] = pipeline
}

// Unregister stops and removes a connection pipeline.
func (r *Registry) Unregister(connID id.ULID) {
	r.mu.Lock()
	pipeline, ok := r.pipelines[connID]
	if ok {
		delete(r.pipelines, connID)
	}
	r.mu.Unlock()

	if ok {
		pipeline.Stop()
	}
}

// Enqueue delivers a payload through the pipeline for a connection.
func (r *Registry) Enqueue(connID id.ULID, payload []byte) error {
	r.mu.RLock()
	pipeline, ok := r.pipelines[connID]
	r.mu.RUnlock()

	if !ok {
		return connection.ErrConnectionNotFound
	}

	return pipeline.Enqueue(payload)
}
