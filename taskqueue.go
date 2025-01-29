package taskqueue

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"time"

	"github.com/oklog/ulid/v2"
)

const DefaultNameSpace = "taskqueue"

type QueueError int

const (
	ErrUnknown QueueError = iota
	ErrQueueNotFound
	ErrQueueEmpty
)

func (err QueueError) Error() string {
	return [...]string{
		ErrUnknown:       "unknown error occurred",
		ErrQueueNotFound: "queue not found",
		ErrQueueEmpty:    "Queue is empty",
	}[err]
}

type JobStatus int8

const (
	JobStatusWaiting JobStatus = iota + 1
	JobStatusActive
	JobStatusCompleted
	JobStatusFailed
	JobStatusDead
)

func (j JobStatus) String() string {
	return []string{
		JobStatusWaiting:   "Waiting",
		JobStatusActive:    "Active",
		JobStatusCompleted: "Completed",
		JobStatusFailed:    "Failed",
		JobStatusDead:      "Dead",
	}[j]
}

var ErrInvalidJobStatus = errors.New("invalid job status")

func ParseJobStatus(text string) (JobStatus, error) {
	switch text {
	case "Waiting":
		return JobStatusWaiting, nil
	case "Active":
		return JobStatusActive, nil
	case "Completed":
		return JobStatusCompleted, nil
	case "Failed":
		return JobStatusFailed, nil
	case "Dead":
		return JobStatusDead, nil
	default:
		return -1, ErrInvalidJobStatus
	}
}

type Job struct {
	ID            string
	QueueName     string
	Payload       []byte
	CreatedAt     time.Time
	StartedAt     time.Time
	UpdatedAt     time.Time
	Attempts      int
	FailureReason string
	Status        JobStatus
	ProcessedBy   string
}

func NewJob() *Job {
	return &Job{
		ID:        ulid.MustNew(ulid.Now(), rand.Reader).String(),
		Status:    JobStatusWaiting,
		CreatedAt: time.Now(),
	}
}

func (j *Job) JSONMarshalPayload(v any) (err error) {
	j.Payload, err = json.Marshal(v)
	return
}

func (j *Job) JSONUnMarshalPayload(v any) error {
	return json.Unmarshal(j.Payload, v)
}

type EnqueueOptions struct {
	QueueName string
	Delay     time.Duration
}

type DequeueOptions struct {
	QueueName  string
	JobTimeout time.Duration
}

type Enqueuer interface {
	Enqueue(ctx context.Context, job *Job, opts *EnqueueOptions) error
}

type Dequeuer interface {
	Dequeue(ctx context.Context, opts *DequeueOptions, count int) ([]*Job, error)
}

type DequeueFunc func(ctx context.Context, opts *DequeueOptions, count int) ([]*Job, error)

func (f DequeueFunc) Dequeue(ctx context.Context, opts *DequeueOptions, count int) ([]*Job, error) {
	return f(ctx, opts, count)
}

type AckOptions struct {
	QueueName string
}

type NackOptions struct {
	QueueName           string
	RetryAfter          time.Duration
	MaxAttemptsExceeded bool
}

type Acker interface {
	Ack(ctx context.Context, job *Job, opts *AckOptions) error
	Nack(ctx context.Context, job *Job, opts *NackOptions) error
}

type QueueStatus int

const (
	QueueStatusUnknown QueueStatus = iota
	QueueStatusPaused
	QueueStatusRunning
)

func (s QueueStatus) String() string {
	return [...]string{
		QueueStatusUnknown: "unknown",
		QueueStatusPaused:  "Paused",
		QueueStatusRunning: "Running",
	}[s]
}

type QueueDetails struct {
	NameSpace  string
	Name       string
	JobCount   int
	Status     QueueStatus
	Pagination Pagination
	Jobs       []*Job
}

type Pagination struct {
	Page int
	Rows int
}

type QueueInfo struct {
	NameSpace string
	Name      string
	JobCount  int
	Status    QueueStatus
}

type QueueManager interface {
	DeleteJobFromDeadQueue(ctx context.Context, queueName string, jobID string) error
	PausePendingQueue(ctx context.Context, queueName string) error
	ResumePendingQueue(ctx context.Context, queueName string) error
	ListPendingQueues(ctx context.Context) ([]*QueueInfo, error)
	ListDeadQueues(ctx context.Context) ([]*QueueInfo, error)
	PagePendingQueue(ctx context.Context, queueName string, p Pagination) (*QueueDetails, error)
	PageDeadQueue(ctx context.Context, queueName string, p Pagination) (*QueueDetails, error)
}

type Queue interface {
	Enqueuer
	Dequeuer
	Acker
	QueueManager
}

var ErrJobNotFound = errors.New("job not found")
