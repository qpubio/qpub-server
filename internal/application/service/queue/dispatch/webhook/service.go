package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	domainJob "github.com/qpubio/qpub-server/internal/domain/queue/job"
	domainQueue "github.com/qpubio/qpub-server/internal/domain/queue/queue"
)

type Service struct {
	client *http.Client
}

func NewService() *Service {
	return &Service{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

type dispatchPayload struct {
	JobID   string          `json:"jobId"`
	Queue   string          `json:"queue"`
	Payload json.RawMessage `json:"payload"`
	Attempt int             `json:"attempt"`
}

func (s *Service) Dispatch(ctx context.Context, q domainQueue.Queue, job domainJob.Job) error {
	if q.WebhookURL == "" {
		return nil
	}

	body, err := json.Marshal(dispatchPayload{
		JobID:   string(job.ID),
		Queue:   job.QueueName,
		Payload: job.Payload,
		Attempt: job.Attempt,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, q.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if q.WebhookSecret != "" {
		mac := hmac.New(sha256.New, []byte(q.WebhookSecret))
		mac.Write(body)
		req.Header.Set("X-QPub-Signature", hex.EncodeToString(mac.Sum(nil)))
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return nil
}
