package protocol

type ActionType int

const (
	ActionJobEnqueued   ActionType = 20
	ActionJobStarted    ActionType = 21
	ActionJobCompleted  ActionType = 22
	ActionJobFailed     ActionType = 23
	ActionJobCancelled  ActionType = 24
)

type JobEvent struct {
	Action   ActionType `json:"action"`
	Queue    string     `json:"queue"`
	JobID    string     `json:"jobId"`
	Status   string     `json:"status"`
	Attempt  int        `json:"attempt,omitempty"`
	Error    string     `json:"error,omitempty"`
}
