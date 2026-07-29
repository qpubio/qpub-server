package dto_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/qpubio/qpub-server/internal/application/dto"
	domainPublication "github.com/qpubio/qpub-server/internal/domain/messaging/publication"

	"github.com/stretchr/testify/require"
)

func TestNewPublishMessageResponse_JSON(t *testing.T) {
	ts := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	res := dto.NewPublishMessageResponse([]domainPublication.PublishResult{
		{
			Channel:      "notifications",
			MessageID:    "01HQABC12345678901234",
			PublishedAt:  ts,
			PayloadCount: 2,
		},
	})

	b, err := json.Marshal(res)
	require.NoError(t, err)

	const want = `{"published":[{"channel":"notifications","message_id":"01HQABC12345678901234","published_at":"2026-05-18T12:00:00Z","payload_count":2}]}`
	require.JSONEq(t, want, string(b))
}
