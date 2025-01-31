package taskmanager

import (
	"time"

	"github.com/oshankkumar/taskqueue-go"
)

type CreateJobRequest struct {
	Args interface{} `json:"args"`
}

type QueueInfo struct {
	NameSpace string `json:"namespace"`
	Name      string `json:"name"`
	JobCount  int    `json:"jobCount"`
	Status    string `json:"status"`
}

type ListQueueResponse struct {
	Queues []QueueInfo `json:"queues"`
}

type ListQueueJobsResponse struct {
	NameSpace  string               `json:"namespace"`
	Name       string               `json:"name"`
	JobCount   int                  `json:"jobCount"`
	Status     string               `json:"status"`
	Pagination taskqueue.Pagination `json:"pagination"`
	Jobs       []Job                `json:"jobs"`
}

type Job struct {
	ID            string      `json:"id"`
	QueueName     string      `json:"queueName"`
	Args          interface{} `json:"args"`
	CreatedAt     time.Time   `json:"createdAt"`
	StartedAt     time.Time   `json:"startedAt"`
	UpdatedAt     time.Time   `json:"updatedAt"`
	Attempts      int         `json:"attempts"`
	FailureReason string      `json:"failureReason"`
	Status        string      `json:"status"`
	ProcessedBy   string      `json:"processedBy"`
}

type ListActiveWorkersResponse struct {
	ActiveWorkers []ActiveWorker `json:"activeWorkers"`
}

type QueuesConfig struct {
	QueueName   string        `json:"queueName"`
	Concurrency int           `json:"concurrency"`
	MaxAttempts int           `json:"maxAttempts"`
	Timeout     time.Duration `json:"timeout"`
}

type ActiveWorker struct {
	WorkerID    string         `json:"workerID"`
	StartedAt   time.Time      `json:"startedAt"`
	HeartbeatAt time.Time      `json:"heartbeatAt"`
	Queues      []QueuesConfig `json:"queues"`
	PID         int            `json:"pid"`
}

type TogglePendingQueueStatusResponse struct {
	NewStatus string `json:"newStatus"`
	OldStatus string `json:"oldStatus"`
}

type MetricsRange struct {
	Metric struct {
		Name   string            `json:"name"`
		Labels map[string]string `json:"labels"`
	} `json:"metric"`
	Values []MetricValue `json:"values"`
}

type MetricValue struct {
	TimeStamp int64   `json:"timestamp"`
	Value     float64 `json:"value"`
}

type MetricsQueryParam struct {
	Name  string
	Start time.Time
	End   time.Time
	Step  time.Duration
}
