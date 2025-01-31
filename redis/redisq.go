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

var (
	//go:embed luascripts/enqueue_inline.lua
	enqueueScriptLuaInline string
	//go:embed luascripts/dequeue_inline.lua
	dequeueScriptLuaInline string
	//go:embed luascripts/ack_inline.lua
	ackScriptLuaInline string
	//go:embed luascripts/retry_inline.lua
	retryScriptLuaInline string
	//go:embed luascripts/move_to_dead_inline.lua
	moveToDeadScriptLuaInline string
)

func NewQueue(client redis.UniversalClient, opts ...OptFunc) *Queue {
	opt := &Options{
		namespace:       taskqueue.DefaultNameSpace,
		completedJobTTL: time.Hour * 24 * 7,
	}
	for _, o := range opts {
		o(opt)
	}
	return &Queue{
		namespace:        opt.namespace,
		completedJobTTL:  opt.completedJobTTL,
		client:           client,
		enqueueScript:    redis.NewScript(enqueueScriptLuaInline),
		dequeueScript:    redis.NewScript(dequeueScriptLuaInline),
		ackScript:        redis.NewScript(ackScriptLuaInline),
		retryScript:      redis.NewScript(retryScriptLuaInline),
		moveToDeadScript: redis.NewScript(moveToDeadScriptLuaInline),
	}
}

type Queue struct {
	namespace        string
	completedJobTTL  time.Duration
	client           redis.UniversalClient
	enqueueScript    *redis.Script
	dequeueScript    *redis.Script
	ackScript        *redis.Script
	retryScript      *redis.Script
	moveToDeadScript *redis.Script
}

func (q *Queue) Enqueue(ctx context.Context, job *taskqueue.Job, opts *taskqueue.EnqueueOptions) error {
	keys := [...]string{
		redisKeyPendingQueue(q.namespace, opts.QueueName),
		redisKeyJob(q.namespace, job.ID),
		redisKeyPendingQueuesSet(q.namespace),
	}
	args := [...]interface{}{
		job.ID,
		time.Now().Add(opts.Delay).Unix(),
		opts.QueueName,
		"id", job.ID,
		"queue_name", opts.QueueName,
		"payload", job.Payload,
		"created_at", job.CreatedAt,
		"started_at", job.StartedAt,
		"updated_at", job.UpdatedAt,
		"attempts", job.Attempts,
		"failure_reason", job.FailureReason,
		"status", JobStatus(job.Status),
		"processed_by", job.ProcessedBy,
	}

	return q.enqueueScript.Run(ctx, q.client, keys[:], args[:]...).Err()
}

func (q *Queue) Dequeue(ctx context.Context, opts *taskqueue.DequeueOptions, count int) ([]*taskqueue.Job, error) {
	var jobs []*taskqueue.Job
	for range count {
		job, err := q.dequeueOne(ctx, opts)
		if err != nil {
			return nil, err
		}
		if job == nil {
			break
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func (q *Queue) dequeueOne(ctx context.Context, opts *taskqueue.DequeueOptions) (*taskqueue.Job, error) {
	keys := [...]string{
		redisKeyPendingQueue(q.namespace, opts.QueueName),
	}
	args := [...]interface{}{
		q.namespace + ":job:",
		time.Now().Unix(),
		int64((opts.JobTimeout + 2*time.Second).Seconds()),
	}

	result, err := q.dequeueScript.Run(ctx, q.client, keys[:], args[:]...).Slice()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	jobRaw, ok := result[1].([]interface{})
	if !ok {
		return nil, fmt.Errorf("dequeueOne: invalid raw job: %v", result[1])
	}

	jobDetails := make(map[string]string)
	for i := 0; i < len(jobRaw); i += 2 {
		field, _ := jobRaw[i].(string)
		value, _ := jobRaw[i+1].(string)
		jobDetails[field] = value
	}

	var redisJob Job
	if err := redis.NewMapStringStringResult(jobDetails, nil).Scan(&redisJob); err != nil {
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

func (q *Queue) Ack(ctx context.Context, job *taskqueue.Job, opts *taskqueue.AckOptions) error {
	keys := [...]string{
		redisKeyPendingQueue(q.namespace, opts.QueueName),
		redisKeyJob(q.namespace, job.ID),
	}
	args := [...]interface{}{
		job.ID,
		int64(q.completedJobTTL.Seconds()),
		"queue_name", opts.QueueName,
		"started_at", job.StartedAt,
		"updated_at", job.UpdatedAt,
		"attempts", job.Attempts,
		"failure_reason", job.FailureReason,
		"status", JobStatus(job.Status),
		"processed_by", job.ProcessedBy,
	}

	return q.ackScript.Run(ctx, q.client, keys[:], args[:]...).Err()
}

func (q *Queue) Nack(ctx context.Context, job *taskqueue.Job, opts *taskqueue.NackOptions) error {
	if opts.ShouldRetry {
		return q.retry(ctx, job, opts)
	}
	return q.moveToDead(ctx, job, opts)
}

func (q *Queue) moveToDead(ctx context.Context, job *taskqueue.Job, opts *taskqueue.NackOptions) error {
	keys := [...]string{
		redisKeyJob(q.namespace, job.ID),
		redisKeyPendingQueue(q.namespace, opts.QueueName),
		redisKeyDeadQueue(q.namespace, opts.QueueName),
		redisKeyDeadQueuesSet(q.namespace),
	}
	args := [...]interface{}{
		job.ID,
		time.Now().Unix(),
		opts.QueueName,
		"queue_name", opts.QueueName,
		"started_at", job.StartedAt,
		"updated_at", job.UpdatedAt,
		"attempts", job.Attempts,
		"failure_reason", job.FailureReason,
		"status", JobStatus(job.Status),
		"processed_by", job.ProcessedBy,
	}

	return q.moveToDeadScript.Run(ctx, q.client, keys[:], args[:]...).Err()
}

func (q *Queue) retry(ctx context.Context, job *taskqueue.Job, opts *taskqueue.NackOptions) error {
	keys := [...]string{
		redisKeyPendingQueue(q.namespace, opts.QueueName),
		redisKeyJob(q.namespace, job.ID),
	}
	args := [...]interface{}{
		job.ID,
		time.Now().Add(opts.RetryAfter).Unix(),
		"queue_name", opts.QueueName,
		"started_at", job.StartedAt,
		"updated_at", job.UpdatedAt,
		"attempts", job.Attempts,
		"failure_reason", job.FailureReason,
		"status", JobStatus(job.Status),
		"processed_by", job.ProcessedBy,
	}

	return q.retryScript.Run(ctx, q.client, keys[:], args[:]...).Err()
}

func (q *Queue) DeleteJobFromDeadQueue(ctx context.Context, queueName string, jobID string) error {
	queueKey := redisKeyDeadQueue(q.namespace, queueName)
	jobKey := redisKeyJob(q.namespace, jobID)

	_, err := q.client.TxPipelined(ctx, func(p redis.Pipeliner) error {
		if err := p.ZRem(ctx, queueKey, jobID).Err(); err != nil {
			return err
		}
		return p.Del(ctx, jobKey).Err()
	})
	if errors.Is(err, redis.Nil) {
		return taskqueue.ErrJobNotFound
	}
	return err
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
	if err != nil {
		return nil, err
	}

	return &taskqueue.QueueInfo{
		NameSpace: q.namespace,
		Name:      queueName,
		JobCount:  int(val),
	}, nil
}

func (q *Queue) PagePendingQueue(ctx context.Context, queueName string, p taskqueue.Pagination) (*taskqueue.QueueDetails, error) {
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
		job, err := q.getJob(ctx, jobID)
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

func (q *Queue) PageDeadQueue(ctx context.Context, queueName string, p taskqueue.Pagination) (*taskqueue.QueueDetails, error) {
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
		job, err := q.getJob(ctx, jobID)
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

func (s *Queue) getJob(ctx context.Context, jobID string) (*taskqueue.Job, error) {
	key := redisKeyJob(s.namespace, jobID)

	var redisJob Job
	if err := s.client.HGetAll(ctx, key).Scan(&redisJob); err != nil {
		return nil, err
	}

	if redisJob.ID == "" {
		return nil, taskqueue.ErrJobNotFound
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
