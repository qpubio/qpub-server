package worker

import "testing"

func TestEncodeQueues(t *testing.T) {
	got := encodeQueues([]string{"emails", "reports"})
	want := `["emails","reports"]`
	if got != want {
		t.Fatalf("encodeQueues() = %q, want %q", got, want)
	}
}

func TestEncodeQueuesEmpty(t *testing.T) {
	if got := encodeQueues(nil); got != "" {
		t.Fatalf("encodeQueues(nil) = %q, want empty", got)
	}
}
