package redis

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/oshankkumar/taskqueue-go"
	"github.com/oshankkumar/taskqueue-go/redisutil"

	"github.com/redis/go-redis/v9"
)

//go:embed dequeue.lua
var dequeueScriptLua string

type JobStorage interface {
	CreateOrUpdate(ctx context.Context, job *taskqueue.Job) error
	GetJob(ctx context.Context, jobID string) (*taskqueue.Job, error)
	DeleteJob(ctx context.Context, jobID string) error
	UpdateJobStatus(ctx context.Context, jobID string, status taskqueue.JobStatus) error
}

func NewQueueWithExternalJobStorage(qclient redis.UniversalClient, s JobStorage, opts ...OptFunc) *QueueWithExternalJobStorage {
	opt := &Options{
		namespace: taskqueue.DefaultNameSpace,
	}
	for _, o := range opts {
		o(opt)
	}
	return &QueueWithExternalJobStorage{
		namespace:     opt.namespace,
		client:        qclient,
		jobStorage:    s,
		dequeueScript: redis.NewScript(dequeueScriptLua),
	}
}

type QueueWithExternalJobStorage struct {
	jobStorage    JobStorage
	namespace     string
	client        redis.UniversalClient
	dequeueScript *redis.Script
}

func (q *QueueWithExternalJobStorage) Enqueue(ctx context.Context, job *taskqueue.Job, opts *taskqueue.EnqueueOptions) error {
	job.QueueName = opts.QueueName

	_, err := q.client.TxPipelined(ctx, func(p redis.Pipeliner) error {
		if err := q.jobStorage.CreateOrUpdate(ctx, job); err != nil {
			return err
		}

		queueKey := redisKeyPendingQueue(q.namespace, opts.QueueName)

		err := p.ZAdd(ctx, queueKey, redis.Z{
			Score:  float64(time.Now().Add(opts.Delay).Unix()),
			Member: job.ID,
		}).Err()
		if err != nil {
			return err
		}

		return q.client.SAdd(ctx, redisKeyPendingQueuesSet(q.namespace), opts.QueueName).Err()
	})

	return err
}

func (q *QueueWithExternalJobStorage) Dequeue(ctx context.Context, opts *taskqueue.DequeueOptions, count int) ([]*taskqueue.Job, error) {
	keys := []string{
		redisKeyPendingQueue(q.namespace, opts.QueueName),
	}
	args := []interface{}{
		time.Now().Unix(),
		int64((opts.JobTimeout + 2*time.Second).Seconds()),
		count,
	}

	jobIDs, err := redisutil.Strings(q.dequeueScript.Run(
		ctx,
		q.client,
		keys,
		args...,
	).Result())
	if errors.Is(err, redis.Nil) {
		return nil, taskqueue.ErrQueueEmpty
	}
	if err != nil {
		return nil, err
	}

	var jobs []*taskqueue.Job
	for _, jobID := range jobIDs {
		job, err := q.jobStorage.GetJob(ctx, jobID)
		if errors.Is(err, taskqueue.ErrJobNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}

		job.Status = taskqueue.JobStatusActive
		if err := q.jobStorage.UpdateJobStatus(ctx, job.ID, job.Status); err != nil {
			return nil, err
		}

		jobs = append(jobs, job)
	}
	return jobs, nil
}

func (q *QueueWithExternalJobStorage) Ack(ctx context.Context, job *taskqueue.Job, opts *taskqueue.AckOptions) error {
	queueKey := redisKeyPendingQueue(q.namespace, opts.QueueName)

	if _, err := q.client.ZRem(ctx, queueKey, job.ID).Result(); err != nil {
		return err
	}

	return q.jobStorage.CreateOrUpdate(ctx, job)
}

func (q *QueueWithExternalJobStorage) Nack(ctx context.Context, job *taskqueue.Job, opts *taskqueue.NackOptions) error {
	if err := q.jobStorage.CreateOrUpdate(ctx, job); err != nil {
		return err
	}

	if opts.MaxAttemptsExceeded {
		return q.nackDead(ctx, job.ID, opts)
	}
	return q.nack(ctx, job.ID, opts)
}

func (q *QueueWithExternalJobStorage) nackDead(ctx context.Context, jobID string, opts *taskqueue.NackOptions) error {
	_, err := q.client.TxPipelined(ctx, func(p redis.Pipeliner) error {
		queueKey := redisKeyPendingQueue(q.namespace, opts.QueueName)
		deadQueueKey := redisKeyDeadQueue(q.namespace, opts.QueueName)

		if err := p.ZRem(ctx, queueKey, jobID).Err(); err != nil {
			return err
		}

		err := p.ZAdd(ctx, deadQueueKey, redis.Z{
			Score:  float64(time.Now().Unix()),
			Member: jobID,
		}).Err()
		if err != nil {
			return err
		}

		return p.SAdd(ctx, redisKeyDeadQueuesSet(q.namespace), opts.QueueName).Err()
	})
	return err
}

func (q *QueueWithExternalJobStorage) nack(ctx context.Context, jobID string, opts *taskqueue.NackOptions) error {
	queueKey := redisKeyPendingQueue(q.namespace, opts.QueueName)

	return q.client.ZAddXX(ctx, queueKey, redis.Z{
		Score:  float64(time.Now().Add(opts.RetryAfter).Unix()),
		Member: jobID,
	}).Err()
}

func (q *QueueWithExternalJobStorage) DeleteJobFromDeadQueue(ctx context.Context, queueName string, jobID string) error {
	queueKey := redisKeyDeadQueue(q.namespace, queueName)
	if err := q.client.ZRem(ctx, queueKey, jobID).Err(); err != nil {
		return err
	}
	return q.jobStorage.DeleteJob(ctx, jobID)
}

func (q *QueueWithExternalJobStorage) PausePendingQueue(ctx context.Context, queueName string) error {
	return q.client.Set(ctx, redisKeyPendingQueuePause(q.namespace, queueName), "1", 0).Err()
}

func (q *QueueWithExternalJobStorage) ResumePendingQueue(ctx context.Context, queueName string) error {
	return q.client.Del(ctx, redisKeyPendingQueuePause(q.namespace, queueName)).Err()
}

func (q *QueueWithExternalJobStorage) ListPendingQueues(ctx context.Context) ([]*taskqueue.QueueInfo, error) {
	queues, err := q.client.SMembers(ctx, redisKeyPendingQueuesSet(q.namespace)).Result()
	if err != nil {
		return nil, err
	}

	var results []*taskqueue.QueueInfo
	for _, queue := range queues {
		info, err := q.pendingQueueInfo(ctx, queue)
		if err != nil {
			return nil, err
		}
		results = append(results, info)
	}

	return results, nil
}

func (q *QueueWithExternalJobStorage) pendingQueueInfo(ctx context.Context, queue string) (*taskqueue.QueueInfo, error) {
	queueKey := redisKeyPendingQueue(q.namespace, queue)

	card, err := q.client.ZCard(ctx, queueKey).Result()
	if errors.Is(err, redis.Nil) {
		return nil, taskqueue.ErrQueueNotFound
	}
	if err != nil {
		return nil, err
	}

	result := &taskqueue.QueueInfo{
		NameSpace: q.namespace,
		Name:      queue,
		JobCount:  int(card),
	}

	val, err := q.client.Exists(ctx, redisKeyPendingQueuePause(q.namespace, queue)).Result()
	if err != nil {
		return nil, err
	}
	if val == 0 {
		result.Status = taskqueue.QueueStatusRunning
	}
	if val == 1 {
		result.Status = taskqueue.QueueStatusPaused
	}

	return result, nil
}

func (q *QueueWithExternalJobStorage) ListDeadQueues(ctx context.Context) ([]*taskqueue.QueueInfo, error) {
	deadQueueSet := redisKeyDeadQueuesSet(q.namespace)
	queues, err := q.client.SMembers(ctx, deadQueueSet).Result()
	if err != nil {
		return nil, err
	}

	var result []*taskqueue.QueueInfo
	for _, queue := range queues {
		info, err := q.deadQueueInfo(ctx, queue)
		if err != nil {
			return nil, err
		}
		result = append(result, info)
	}
	return result, nil
}

func (q *QueueWithExternalJobStorage) deadQueueInfo(ctx context.Context, queueName string) (*taskqueue.QueueInfo, error) {
	queueKey := redisKeyDeadQueue(q.namespace, queueName)

	val, err := q.client.ZCard(ctx, queueKey).Result()
	if errors.Is(err, redis.Nil) {
		return nil, taskqueue.ErrQueueNotFound
	}
	if err != nil {
		return nil, err
	}

	return &taskqueue.QueueInfo{
		NameSpace: q.namespace,
		Name:      queueName,
		JobCount:  int(val),
	}, nil
}

func (q *QueueWithExternalJobStorage) PagePendingQueue(ctx context.Context, queueName string, p taskqueue.Pagination) (*taskqueue.QueueDetails, error) {
	info, err := q.pendingQueueInfo(ctx, queueName)
	if err != nil {
		return nil, err
	}

	queueKey := redisKeyPendingQueue(q.namespace, queueName)
	offset := (p.Page - 1) * p.Rows

	jobIDs, err := q.client.ZRangeArgs(ctx, redis.ZRangeArgs{
		Key:     queueKey,
		Start:   "-inf",
		Stop:    "+inf",
		ByScore: true,
		Offset:  int64(offset),
		Count:   int64(p.Rows),
	}).Result()
	if err != nil {
		return nil, err
	}

	var jobs []*taskqueue.Job
	for _, jobID := range jobIDs {
		job, err := q.jobStorage.GetJob(ctx, jobID)
		if errors.Is(err, taskqueue.ErrJobNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	return &taskqueue.QueueDetails{
		NameSpace:  q.namespace,
		Name:       queueName,
		JobCount:   info.JobCount,
		Status:     info.Status,
		Pagination: p,
		Jobs:       jobs,
	}, nil
}

func (q *QueueWithExternalJobStorage) PageDeadQueue(ctx context.Context, queueName string, p taskqueue.Pagination) (*taskqueue.QueueDetails, error) {
	info, err := q.deadQueueInfo(ctx, queueName)
	if err != nil {
		return nil, err
	}

	queueKey := redisKeyDeadQueue(q.namespace, queueName)
	offset := (p.Page - 1) * p.Rows

	jobIDs, err := q.client.ZRangeArgs(ctx, redis.ZRangeArgs{
		Key:     queueKey,
		Start:   "-inf",
		Stop:    "+inf",
		ByScore: true,
		Offset:  int64(offset),
		Count:   int64(p.Rows),
	}).Result()
	if err != nil {
		return nil, err
	}

	var jobs []*taskqueue.Job
	for _, jobID := range jobIDs {
		job, err := q.jobStorage.GetJob(ctx, jobID)
		if errors.Is(err, taskqueue.ErrJobNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	return &taskqueue.QueueDetails{
		NameSpace:  q.namespace,
		Name:       queueName,
		JobCount:   info.JobCount,
		Pagination: p,
		Jobs:       jobs,
	}, nil
}

func redisKeyDeadQueuesSet(ns string) string {
	return ns + ":dead-queues"
}

func redisKeyPendingQueuesSet(ns string) string {
	return ns + ":pending-queues"
}

func redisKeyPendingQueue(ns string, queue string) string {
	return fmt.Sprintf("%s:queue:%s:pending", ns, queue)
}

func redisKeyDeadQueue(ns string, queue string) string {
	return fmt.Sprintf("%s:queue:%s:dead", ns, queue)
}

func redisKeyPendingQueuePause(ns string, queue string) string {
	return redisKeyPendingQueue(ns, queue) + ":pause"
}
