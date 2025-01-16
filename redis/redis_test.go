package redis

import (
	"context"
	"crypto/rand"
	"os"
	"testing"
	"time"

	"github.com/oshankkumar/taskqueue-go"

	"github.com/google/go-cmp/cmp"
	"github.com/redis/go-redis/v9"
)

func TestStoreCreateOrUpdate(t *testing.T) {
	t.Setenv("REDIS_ADDR", "localhost:7379")
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		t.Skip("skipping test since REDIS_ADDR is not set")
	}

	client := redis.NewClient(&redis.Options{Addr: redisAddr})
	store := NewStore(client, "test")

	var payload [32]byte
	if _, err := rand.Read(payload[:]); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2025, 1, 1, 1, 10, 0, 0, time.UTC)

	job := &taskqueue.Job{
		ID:            "test-id1",
		QueueName:     "test_queue",
		Payload:       payload[:],
		CreatedAt:     now,
		StartedAt:     now,
		UpdatedAt:     now,
		Attempts:      2,
		FailureReason: taskqueue.ErrJobNotFound,
		Status:        taskqueue.JobStatusActive,
		ProcessedBy:   "test-worker-1",
	}

	if err := store.CreateOrUpdate(context.Background(), job); err != nil {
		t.Fatal(err)
	}

	got, err := store.GetJob(context.Background(), job.ID)
	if err != nil {
		t.Fatal(err)
	}

	equateErrorMessage := cmp.Comparer(func(x, y error) bool {
		if x == nil || y == nil {
			return x == nil && y == nil
		}
		return x.Error() == y.Error()
	})

	if !cmp.Equal(job, got, equateErrorMessage) {
		t.Errorf("job does not match the expected one. Diff:\n%s", cmp.Diff(job, got))
	}
}

func TestStoreLastHeartbeats(t *testing.T) {
	t.Setenv("REDIS_ADDR", "localhost:7379")
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		t.Skip("skipping test since REDIS_ADDR is not set")
	}

	client := redis.NewClient(&redis.Options{Addr: redisAddr})
	store := NewHeartBeater(client, "test")

	hb := taskqueue.HeartbeatData{
		WorkerID:    "12345",
		StartedAt:   time.Now().UTC(),
		HeartbeatAt: time.Now().UTC(),
		Queues: []taskqueue.HeartbeatQueueData{
			{
				Name:        "queue_1",
				Concurrency: 10,
				MaxAttempts: 10,
				Timeout:     time.Minute * 10,
			},
			{
				Name:        "queue_2",
				Concurrency: 10,
				MaxAttempts: 10,
				Timeout:     time.Minute,
			},
			{
				Name:        "queue_2",
				Concurrency: 10,
				MaxAttempts: 10,
				Timeout:     time.Second * 30,
			},
		},
		PID: 12,
	}
	if err := store.SendHeartbeat(context.Background(), hb); err != nil {
		t.Fatal(err)
	}

	hbs, err := store.LastHeartbeats(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if !cmp.Equal(hbs[0], hb) {
		t.Fatal(cmp.Diff(hbs[0], hb))
	}
}
