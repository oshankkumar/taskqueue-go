package redis

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/oshankkumar/taskqueue-go"
	"github.com/redis/go-redis/v9"
)

type HeartbeatData struct {
	WorkerID    string    `redis:"worker_id"`
	StartedAt   time.Time `redis:"started_at"`
	HeartbeatAt time.Time `redis:"heartbeat_at"`
	Queues      []byte    `redis:"queues"`
	PID         int       `redis:"pid"`
	MemoryUsage float64   `redis:"memory_usage"`
	CPUUsage    float64   `redis:"cpu_usage"`
}

func NewHeartBeater(rc redis.UniversalClient, opts ...OptFunc) *Heartbeater {
	opt := &Options{
		namespace: taskqueue.DefaultNameSpace,
	}
	for _, o := range opts {
		o(opt)
	}
	return &Heartbeater{namespace: opt.namespace, client: rc}
}

type Heartbeater struct {
	namespace string
	client    redis.UniversalClient
}

func (s *Heartbeater) SendHeartbeat(ctx context.Context, data taskqueue.HeartbeatData) error {
	workersSetKey := redisKeyWorkersSet(s.namespace)
	workerKey := redisKeyWorkerHeartbeat(s.namespace, data.WorkerID)

	queuesData, err := json.Marshal(data.Queues)
	if err != nil {
		return err
	}
	heartbeatData := HeartbeatData{
		WorkerID:    data.WorkerID,
		StartedAt:   data.StartedAt,
		HeartbeatAt: data.HeartbeatAt,
		Queues:      queuesData,
		PID:         data.PID,
		MemoryUsage: data.MemoryUsage,
		CPUUsage:    data.CPUUsage,
	}

	_, err = s.client.TxPipelined(ctx, func(p redis.Pipeliner) error {
		if err := p.SAdd(ctx, workersSetKey, data.WorkerID).Err(); err != nil {
			return err
		}

		return p.HSet(ctx, workerKey, heartbeatData).Err()
	})

	return err
}

func (s *Heartbeater) RemoveHeartbeat(ctx context.Context, workerID string) error {
	workersSetKey := redisKeyWorkersSet(s.namespace)
	workerKey := redisKeyWorkerHeartbeat(s.namespace, workerID)

	_, err := s.client.TxPipelined(ctx, func(p redis.Pipeliner) error {
		if err := p.SRem(ctx, workersSetKey, workerID).Err(); err != nil {
			return err
		}
		return p.Del(ctx, workerKey).Err()
	})

	return err
}

func (s *Heartbeater) LastHeartbeats(ctx context.Context) ([]taskqueue.HeartbeatData, error) {
	workersSetKey := redisKeyWorkersSet(s.namespace)

	workerIDs, err := s.client.SMembers(ctx, workersSetKey).Result()
	if err != nil {
		return nil, err
	}

	var result []taskqueue.HeartbeatData

	for _, workerID := range workerIDs {
		workerKey := redisKeyWorkerHeartbeat(s.namespace, workerID)
		var hb HeartbeatData
		err := s.client.HGetAll(ctx, workerKey).Scan(&hb)
		if errors.Is(err, redis.Nil) {
			result = append(result, taskqueue.HeartbeatData{WorkerID: workerID})
			continue
		}
		if err != nil {
			return nil, err
		}

		var queues []taskqueue.HeartbeatQueueData
		if err := json.Unmarshal(hb.Queues, &queues); err != nil {
			return nil, err
		}

		result = append(result, taskqueue.HeartbeatData{
			WorkerID:    hb.WorkerID,
			StartedAt:   hb.StartedAt,
			HeartbeatAt: hb.HeartbeatAt,
			Queues:      queues,
			PID:         hb.PID,
			MemoryUsage: hb.MemoryUsage,
			CPUUsage:    hb.CPUUsage,
		})
	}

	return result, nil
}
