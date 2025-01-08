package redis

import (
	"context"
	"errors"
	"fmt"

	"github.com/oshankkumar/taskqueue-go"

	"github.com/redis/go-redis/v9"
)

func NewStore(client redis.UniversalClient, ns string) *Store {
	if ns == "" {
		ns = taskqueue.DefaultNameSpace
	}
	return &Store{Namespace: ns, client: client}
}

type Store struct {
	Namespace string
	client    redis.UniversalClient
}

func (s *Store) CreateOrUpdate(ctx context.Context, job *taskqueue.Job) error {
	key := redisJobKey(s.Namespace, job.ID)

	redisJob := Job{
		ID:            job.ID,
		QueueName:     job.QueueName,
		Payload:       job.Payload,
		CreatedAt:     job.CreatedAt,
		StartedAt:     job.StartedAt,
		UpdatedAt:     job.UpdatedAt,
		Attempts:      job.Attempts,
		Status:        JobStatus(job.Status),
		FailureReason: JobError{Err: job.FailureReason},
		ProcessedBy:   job.ProcessedBy,
	}

	_, err := s.client.TxPipelined(ctx, func(p redis.Pipeliner) error {
		_, err := p.HSet(ctx, key, redisJob).Result()
		if err != nil {
			return err
		}

		if job.Status == taskqueue.JobStatusDead {
			return p.SAdd(ctx, redisDeadJobsSetKey(s.Namespace), job.ID).Err()
		}

		return nil
	})

	return err
}

func (s *Store) GetJob(ctx context.Context, jobID string) (*taskqueue.Job, error) {
	key := redisJobKey(s.Namespace, jobID)

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
		FailureReason: redisJob.FailureReason.Err,
		Status:        taskqueue.JobStatus(redisJob.Status),
		ProcessedBy:   redisJob.ProcessedBy,
	}, nil
}

func (s *Store) DeleteJob(ctx context.Context, jobID string) error {
	key := redisJobKey(s.Namespace, jobID)

	err := s.client.Del(ctx, key).Err()
	if errors.Is(err, redis.Nil) {
		return taskqueue.ErrJobNotFound
	}
	return err
}

func (s *Store) UpdateJobStatus(ctx context.Context, jobID string, status taskqueue.JobStatus) error {
	key := redisJobKey(s.Namespace, jobID)

	err := s.client.HSet(ctx, key, "status", status.String()).Err()
	if errors.Is(err, redis.Nil) {
		return taskqueue.ErrJobNotFound
	}
	return err
}

func redisJobKey(ns string, jobID string) string {
	return fmt.Sprintf("%s:job:%s", ns, jobID)
}

func redisDeadJobsSetKey(ns string) string {
	return fmt.Sprintf("%s:dead-jobs", ns)
}
