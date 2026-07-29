package backpressure_test

import (
	"testing"

	"github.com/qpubio/qpub-server/internal/domain/messaging/backpressure"
	"github.com/qpubio/qpub-server/internal/domain/messaging/testsupport"
	"github.com/qpubio/qpub-server/internal/shared/id"

	"github.com/stretchr/testify/require"
)

func TestFixedWindowLimiter_AllowsUpToLimit(t *testing.T) {
	testsupport.InitRuntime()
	limiter := backpressure.NewFixedWindowLimiter(2)
	require.True(t, limiter.Allow())
	require.True(t, limiter.Allow())
	require.False(t, limiter.Allow())
}

func TestProjectLimiter_RefreshesWhenLimitChanges(t *testing.T) {
	testsupport.InitRuntime()

	projects := backpressure.NewProjectLimiter()
	projectID := id.Int(9)

	require.True(t, projects.Allow(projectID, 1))
	require.False(t, projects.Allow(projectID, 1))

	require.True(t, projects.Allow(projectID, 2))
	require.True(t, projects.Allow(projectID, 2))
}
