package redis

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/oshankkumar/taskqueue-go"

	"github.com/redis/go-redis/v9"
)

//go:embed dequeue.lua
var dequeueLuaScript string

func NewQueue(client redis.UniversalClient, ns string) *Queue {
	if ns == "" {
		ns = taskqueue.DefaultNameSpace
	}
	return &Queue{
		namespace:     ns,
		client:        client,
		dequeueScript: redis.NewScript(dequeueLuaScript),
	}
}

type Queue struct {
	namespace     string
	client        redis.UniversalClient
	dequeueScript *redis.Script
}

func (q *Queue) Enqueue(ctx context.Context, jobID string, opts *taskqueue.EnqueueOptions) error {
	_, err := q.client.TxPipelined(ctx, func(p redis.Pipeliner) error {
		queueKey := redisQueueKey(q.namespace, opts.QueueName)

		err := p.ZAdd(ctx, queueKey, redis.Z{
			Score:  float64(time.Now().Unix()),
			Member: jobID,
		}).Err()
		if err != nil {
			return err
		}

		return q.client.SAdd(ctx, redisKeyQueuesSet(q.namespace), queueKey).Err()
	})

	return err
}

func (q *Queue) Dequeue(ctx context.Context, opts *taskqueue.DequeueOptions, count int) ([]string, error) {
	keys := []string{
		redisQueueKey(q.namespace, opts.QueueName),
	}
	args := []interface{}{
		time.Now().Unix(),
		int64((opts.JobTimeout + 2*time.Second).Seconds()),
		count,
	}

	jobIDs, err := Strings(q.dequeueScript.Run(
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
	queueKey := redisQueueKey(q.namespace, opts.QueueName)
	_, err := q.client.ZRem(ctx, queueKey, jobID).Result()
	return err
}

func (q *Queue) Nack(ctx context.Context, jobID string, opts *taskqueue.NackOptions) error {
	queueKey := redisQueueKey(q.namespace, opts.QueueName)

	_, err := q.client.ZAddXX(ctx, queueKey, redis.Z{
		Score:  float64(time.Now().Add(opts.RetryAfter).Unix()),
		Member: jobID,
	}).Result()

	return err
}

func redisQueueKey(ns string, queue string) string {
	return fmt.Sprintf("%s:queue:%s", ns, queue)
}

func redisKeyQueuesSet(ns string) string {
	return ns + ":queues"
}

func Strings(i interface{}, err error) ([]string, error) {
	if err != nil {
		return nil, err
	}

	vv, ok := i.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid type: %T expected to be a []interface{}", i)
	}

	ss := make([]string, 0, len(vv))
	for _, v := range vv {
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("invalid type: %T expected to be a string", v)
		}
		ss = append(ss, s)
	}

	return ss, nil
}
