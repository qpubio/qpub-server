package task

// TaskName defines task identifiers
type TaskName string

// TaskType indicates how a task should be distributed
type TaskType int

const (
	TaskTypeSingle TaskType = iota
	TaskTypePerInstance
)

const (
	TaskStatsCollectMinutely       TaskName = "stats:collect:minutely"
	TaskStatsCleanupDaily          TaskName = "stats:cleanup:daily"
	TaskProjectUsageGenerateHourly TaskName = "project:usage:generate:hourly"
	TaskProjectUsageCleanupDaily   TaskName = "project:usage:cleanup:daily"
	TaskInstanceCleanupMinutely    TaskName = "instance:cleanup:minutely"
)
