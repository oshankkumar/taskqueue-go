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
var dequeueLuaScript string

func NewQueue(client redis.UniversalClient, opts ...OptFunc) *Queue {
	opt := &Options{
		namespace: taskqueue.DefaultNameSpace,
	}
	for _, o := range opts {
		o(opt)
	}
	return &Queue{
		namespace:     opt.namespace,
		client:        client,
		dequeueScript: redis.NewScript(dequeueLuaScript),
	}
}

type Queue struct {
	namespace     string
	client        redis.UniversalClient
	dequeueScript *redis.Script
}

func (q *Queue) DeleteJobFromDeadQueue(ctx context.Context, queueName string, jobID string) error {
	queueKey := redisKeyDeadQueue(q.namespace, queueName)
	return q.client.ZRem(ctx, queueKey, jobID).Err()
}

func (q *Queue) ListDeadQueues(ctx context.Context) ([]*taskqueue.QueueInfo, error) {
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

func (q *Queue) deadQueueInfo(ctx context.Context, queueName string) (*taskqueue.QueueInfo, error) {
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

func (q *Queue) PageDeadQueue(ctx context.Context, queueName string, p taskqueue.Pagination) (*taskqueue.QueueDetails, error) {
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

	info, err := q.deadQueueInfo(ctx, queueName)
	if err != nil {
		return nil, err
	}

	return &taskqueue.QueueDetails{
		NameSpace:  q.namespace,
		Name:       queueName,
		JobCount:   info.JobCount,
		Pagination: p,
		JobIDs:     jobIDs,
	}, nil

}

func (q *Queue) PausePendingQueue(ctx context.Context, queueName string) error {
	return q.client.Set(ctx, redisKeyPendingQueuePause(q.namespace, queueName), "1", 0).Err()
}

func (q *Queue) ResumePendingQueue(ctx context.Context, queueName string) error {
	return q.client.Del(ctx, redisKeyPendingQueuePause(q.namespace, queueName)).Err()
}

func (q *Queue) ListPendingQueues(ctx context.Context) ([]*taskqueue.QueueInfo, error) {
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

func (q *Queue) pendingQueueInfo(ctx context.Context, queue string) (*taskqueue.QueueInfo, error) {
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

func (q *Queue) PagePendingQueue(ctx context.Context, queueName string, p taskqueue.Pagination) (*taskqueue.QueueDetails, error) {
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

	info, err := q.pendingQueueInfo(ctx, queueName)
	if err != nil {
		return nil, err
	}

	return &taskqueue.QueueDetails{
		NameSpace:  q.namespace,
		Name:       queueName,
		JobCount:   info.JobCount,
		Status:     info.Status,
		Pagination: p,
		JobIDs:     jobIDs,
	}, nil
}

func (q *Queue) Enqueue(ctx context.Context, jobID string, opts *taskqueue.EnqueueOptions) error {
	_, err := q.client.TxPipelined(ctx, func(p redis.Pipeliner) error {
		queueKey := redisKeyPendingQueue(q.namespace, opts.QueueName)

		err := p.ZAdd(ctx, queueKey, redis.Z{
			Score:  float64(time.Now().Add(opts.Delay).Unix()),
			Member: jobID,
		}).Err()
		if err != nil {
			return err
		}

		return q.client.SAdd(ctx, redisKeyPendingQueuesSet(q.namespace), opts.QueueName).Err()
	})

	return err
}

func (q *Queue) Dequeue(ctx context.Context, opts *taskqueue.DequeueOptions, count int) ([]string, error) {
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

	return jobIDs, nil
}

func (q *Queue) Ack(ctx context.Context, jobID string, opts *taskqueue.AckOptions) error {
	queueKey := redisKeyPendingQueue(q.namespace, opts.QueueName)
	_, err := q.client.ZRem(ctx, queueKey, jobID).Result()
	return err
}

func (q *Queue) Nack(ctx context.Context, jobID string, opts *taskqueue.NackOptions) error {
	if opts.MaxAttemptsExceeded {
		return q.nackDead(ctx, jobID, opts)
	}
	return q.nack(ctx, jobID, opts)
}

func (q *Queue) nackDead(ctx context.Context, jobID string, opts *taskqueue.NackOptions) error {
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

func (q *Queue) nack(ctx context.Context, jobID string, opts *taskqueue.NackOptions) error {
	queueKey := redisKeyPendingQueue(q.namespace, opts.QueueName)

	return q.client.ZAddXX(ctx, queueKey, redis.Z{
		Score:  float64(time.Now().Add(opts.RetryAfter).Unix()),
		Member: jobID,
	}).Err()
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
