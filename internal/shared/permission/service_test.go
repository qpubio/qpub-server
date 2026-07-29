package permission_test

import (
	"testing"

	"github.com/qpubio/qpub-server/internal/shared/permission"
)

func TestCanSubscribe_PlatformChannelsRequireDedicatedActions(t *testing.T) {
	svc := permission.NewService()

	tests := []struct {
		name     string
		perm     string
		resource string
		want     bool
	}{
		{
			name:     "subscribe alone cannot access _stats",
			perm:     `{"*":["subscribe"]}`,
			resource: "_stats",
			want:     false,
		},
		{
			name:     "subscribe alone cannot access _logs",
			perm:     `{"*":["subscribe"]}`,
			resource: "_logs",
			want:     false,
		},
		{
			name:     "stats grants _stats",
			perm:     `{"*":["stats"]}`,
			resource: "_stats",
			want:     true,
		},
		{
			name:     "logs grants _logs",
			perm:     `{"*":["logs"]}`,
			resource: "_logs",
			want:     true,
		},
		{
			name:     "stats does not grant _logs",
			perm:     `{"*":["stats"]}`,
			resource: "_logs",
			want:     false,
		},
		{
			name:     "logs does not grant _stats",
			perm:     `{"*":["logs"]}`,
			resource: "_stats",
			want:     false,
		},
		{
			name:     "wildcard action grants platform channels",
			perm:     `{"*":["*"]}`,
			resource: "_stats",
			want:     true,
		},
		{
			name:     "subscribe still grants normal channels",
			perm:     `{"*":["subscribe"]}`,
			resource: "notifications",
			want:     true,
		},
		{
			name:     "stats does not grant normal channels",
			perm:     `{"*":["stats"]}`,
			resource: "notifications",
			want:     false,
		},
		{
			name:     "resource-scoped stats grants _stats",
			perm:     `{"_stats":["stats"]}`,
			resource: "_stats",
			want:     true,
		},
		{
			name:     "resource-scoped logs grants _logs",
			perm:     `{"_logs":["logs"]}`,
			resource: "_logs",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.CanSubscribe([]byte(tt.perm), tt.resource)
			if err != nil {
				t.Fatalf("CanSubscribe() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("CanSubscribe() = %v, want %v", got, tt.want)
			}
		})
	}
}
