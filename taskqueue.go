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
	ErrQueueEmpty
)

func (err QueueError) Error() string {
	return [...]string{
		ErrUnknown:    "unknown error occurred",
		ErrQueueEmpty: "Queue is empty",
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
		JobStatusWaiting:   "waiting",
		JobStatusActive:    "active",
		JobStatusCompleted: "completed",
		JobStatusFailed:    "failed",
		JobStatusDead:      "dead",
	}[j]
}

var ErrInvalidJobStatus = errors.New("invalid job status")

func ParseJobStatus(text string) (JobStatus, error) {
	switch text {
	case "waiting":
		return JobStatusWaiting, nil
	case "active":
		return JobStatusActive, nil
	case "completed":
		return JobStatusCompleted, nil
	case "failed":
		return JobStatusFailed, nil
	case "dead":
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
	FailureReason error
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
}

type DequeueOptions struct {
	QueueName  string
	JobTimeout time.Duration
}

type Enqueuer interface {
	Enqueue(ctx context.Context, jobID string, opts *EnqueueOptions) error
}

type Dequeuer interface {
	Dequeue(ctx context.Context, opts *DequeueOptions, count int) ([]string, error)
}

type DequeueFunc func(ctx context.Context, opts *DequeueOptions, count int) ([]string, error)

func (f DequeueFunc) Dequeue(ctx context.Context, opts *DequeueOptions, count int) ([]string, error) {
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
	Ack(ctx context.Context, jobID string, opts *AckOptions) error
	Nack(ctx context.Context, jobID string, opts *NackOptions) error
}

type QueueStatus int

const (
	QueueStatusUnknown QueueStatus = iota
	QueueStatusPaused
	QueueStatusRunning
)

type QueueDetails struct {
	NameSpace  string
	Name       string
	JobCount   int
	Status     QueueStatus
	Pagination Pagination
	JobIDs     []string
}

type Pagination struct {
	Page      int
	RowsCount int
}

type QueueInfo struct {
	NameSpace string
	Name      string
	JobCount  int
	Status    QueueStatus
}

type QueueManager interface {
	PauseQueue(ctx context.Context, queueName string) error
	ResumeQueue(ctx context.Context, queueName string) error
	ListPendingQueues(ctx context.Context) ([]*QueueInfo, error)
	PendingQueueInfo(ctx context.Context, queueName string) (*QueueInfo, error)
	PagePendingQueue(ctx context.Context, queueName string, p Pagination) (*QueueDetails, error)
}

type Queue interface {
	Enqueuer
	Dequeuer
	Acker
	QueueManager
}

var ErrJobNotFound = errors.New("job not found")

type JobStore interface {
	CreateOrUpdate(ctx context.Context, job *Job) error
	GetJob(ctx context.Context, jobID string) (*Job, error)
	DeleteJob(ctx context.Context, jobID string) error
	UpdateJobStatus(ctx context.Context, jobID string, status JobStatus) error
}

type TaskEnqueuer struct {
	Enqueuer Enqueuer
	JobStore JobStore
}

func (t *TaskEnqueuer) Enqueue(ctx context.Context, job *Job, opts *EnqueueOptions) error {
	job.QueueName = opts.QueueName

	if err := t.JobStore.CreateOrUpdate(ctx, job); err != nil {
		return err
	}

	return t.Enqueuer.Enqueue(ctx, job.ID, opts)
}
