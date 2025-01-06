package redis

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

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

	var failureReason string
	if job.FailureReason != nil {
		failureReason = job.FailureReason.Error()
	}

	fields := map[string]interface{}{
		"id":             job.ID,
		"queue_name":     job.QueueName,
		"payload":        job.Payload,
		"status":         job.Status.String(),
		"created_at":     job.CreatedAt.Format(time.RFC3339),
		"updated_at":     job.UpdatedAt.Format(time.RFC3339),
		"started_at":     job.StartedAt.Format(time.RFC3339),
		"attempts":       job.Attempts,
		"failure_reason": failureReason,
		"processed_by":   job.ProcessedBy,
	}

	_, err := s.client.HSet(ctx, key, fields).Result()
	if err != nil {
		return err
	}

	if job.Status == taskqueue.JobStatusDead {
		return s.client.SAdd(ctx, redisDeadJobsSetKey(s.Namespace), job.ID).Err()
	}

	return nil
}

func (s *Store) GetJob(ctx context.Context, jobID string) (*taskqueue.Job, error) {
	key := redisJobKey(s.Namespace, jobID)

	vals, err := s.client.HGetAll(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return nil, taskqueue.ErrJobNotFound
	}
	if err != nil {
		return nil, err
	}

	createdAt, err := time.Parse(time.RFC3339, vals["created_at"])
	if err != nil {
		return nil, err
	}

	updatedAt, err := time.Parse(time.RFC3339, vals["updated_at"])
	if err != nil {
		return nil, err
	}

	startedAt, err := time.Parse(time.RFC3339, vals["started_at"])
	if err != nil {
		return nil, err
	}

	attempts, err := strconv.Atoi(vals["attempts"])
	if err != nil {
		return nil, err
	}

	var failureReason error
	if vals["failure_reason"] != "" {
		failureReason = errors.New(vals["failure_reason"])
	}

	status, err := taskqueue.ParseJobStatus(vals["status"])
	if err != nil {
		return nil, err
	}

	return &taskqueue.Job{
		ID:            vals["id"],
		QueueName:     vals["queue_name"],
		Payload:       []byte(vals["payload"]),
		CreatedAt:     createdAt,
		StartedAt:     startedAt,
		UpdatedAt:     updatedAt,
		Attempts:      attempts,
		FailureReason: failureReason,
		Status:        status,
		ProcessedBy:   vals["processed_by"],
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
