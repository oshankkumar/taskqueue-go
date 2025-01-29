package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/oshankkumar/taskqueue-go"
	"github.com/redis/go-redis/v9"
)

type JobStatus int8

func (j *JobStatus) ScanRedis(s string) error {
	st, err := taskqueue.ParseJobStatus(s)
	if err != nil {
		return err
	}

	*j = JobStatus(st)
	return nil
}

func (j JobStatus) MarshalBinary() (data []byte, err error) {
	return []byte(taskqueue.JobStatus(j).String()), nil
}

type Job struct {
	ID            string    `redis:"id"`
	QueueName     string    `redis:"queue_name"`
	Payload       []byte    `redis:"payload"`
	CreatedAt     time.Time `redis:"created_at"`
	StartedAt     time.Time `redis:"started_at"`
	UpdatedAt     time.Time `redis:"updated_at"`
	Attempts      int       `redis:"attempts"`
	FailureReason string    `redis:"failure_reason"`
	Status        JobStatus `redis:"status"`
	ProcessedBy   string    `redis:"processed_by"`
}

func NewStore(client redis.UniversalClient, opts ...OptFunc) *Store {
	opt := &Options{
		namespace: taskqueue.DefaultNameSpace,
	}
	for _, o := range opts {
		o(opt)
	}

	return &Store{namespace: opt.namespace, client: client, completedJobTTL: opt.completedJobTTL}
}

type Store struct {
	namespace       string
	client          redis.UniversalClient
	completedJobTTL time.Duration
}

func (s *Store) CreateOrUpdate(ctx context.Context, job *taskqueue.Job) error {
	key := redisKeyJob(s.namespace, job.ID)

	redisJob := Job{
		ID:            job.ID,
		QueueName:     job.QueueName,
		Payload:       job.Payload,
		CreatedAt:     job.CreatedAt,
		StartedAt:     job.StartedAt,
		UpdatedAt:     job.UpdatedAt,
		Attempts:      job.Attempts,
		Status:        JobStatus(job.Status),
		FailureReason: job.FailureReason,
		ProcessedBy:   job.ProcessedBy,
	}

	return s.client.HSet(ctx, key, redisJob).Err()
}

func (s *Store) GetJob(ctx context.Context, jobID string) (*taskqueue.Job, error) {
	key := redisKeyJob(s.namespace, jobID)

	var redisJob Job
	err := s.client.HGetAll(ctx, key).Scan(&redisJob)
	if errors.Is(err, redis.Nil) {
		return nil, taskqueue.ErrJobNotFound
	}
	if err != nil {
		return nil, err
	}

	return &taskqueue.Job{
		ID:            redisJob.ID,
		QueueName:     redisJob.QueueName,
		Payload:       redisJob.Payload,
		CreatedAt:     redisJob.CreatedAt,
		StartedAt:     redisJob.StartedAt,
		UpdatedAt:     redisJob.UpdatedAt,
		Attempts:      redisJob.Attempts,
		FailureReason: redisJob.FailureReason,
		Status:        taskqueue.JobStatus(redisJob.Status),
		ProcessedBy:   redisJob.ProcessedBy,
	}, nil
}

func (s *Store) DeleteJob(ctx context.Context, jobID string) error {
	key := redisKeyJob(s.namespace, jobID)

	err := s.client.Del(ctx, key).Err()
	if errors.Is(err, redis.Nil) {
		return taskqueue.ErrJobNotFound
	}
	return err
}

func (s *Store) UpdateJobStatus(ctx context.Context, jobID string, status taskqueue.JobStatus) error {
	key := redisKeyJob(s.namespace, jobID)

	_, err := s.client.TxPipelined(ctx, func(p redis.Pipeliner) error {
		err := p.HSet(ctx, key, "status", status.String()).Err()
		if errors.Is(err, redis.Nil) {
			return taskqueue.ErrJobNotFound
		}

		if err != nil {
			return err
		}

		if err := p.HSet(ctx, key, "updated_at", time.Now()).Err(); err != nil {
			return err
		}

		if status == taskqueue.JobStatusCompleted && s.completedJobTTL > 0 {
			return p.Expire(ctx, key, s.completedJobTTL).Err()
		}
		return nil
	})

	return err
}

func redisKeyJob(ns string, jobID string) string {
	return fmt.Sprintf("%s:job:%s", ns, jobID)
}

func redisKeyWorkersSet(ns string) string {
	return fmt.Sprintf("%s:workers", ns)
}

func redisKeyWorkerHeartbeat(ns string, workerID string) string {
	return fmt.Sprintf("%s:worker:%s", ns, workerID)
}
