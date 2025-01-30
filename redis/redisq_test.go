package redis

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/oshankkumar/taskqueue-go"

	"github.com/redis/go-redis/v9"
)

func TestInlineQueueEnqueue(t *testing.T) {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		t.Skip("skipping test since REDIS_ADDR is not set")
	}

	client := redis.NewClient(&redis.Options{Addr: redisAddr})

	q := NewInlineQueue(client)

	var payload [32]byte
	if _, err := rand.Read(payload[:]); err != nil {
		t.Fatal(err)
	}

	now := time.Now()

	job := &taskqueue.Job{
		ID:            "test-inline-job-1",
		QueueName:     "test_inline_queue",
		Payload:       payload[:],
		CreatedAt:     now,
		StartedAt:     now,
		UpdatedAt:     now,
		Attempts:      2,
		FailureReason: taskqueue.ErrJobNotFound.Error(),
		Status:        taskqueue.JobStatusWaiting,
		ProcessedBy:   "test-worker-1",
	}

	err := q.Enqueue(context.Background(), job, &taskqueue.EnqueueOptions{QueueName: "test_inline_queue"})
	if err != nil {
		t.Fatal(err)
	}

	jobs, err := q.Dequeue(context.Background(), &taskqueue.DequeueOptions{QueueName: "test_inline_queue"}, 1)
	if err != nil {
		t.Fatal(err)
	}

	if len(jobs) != 1 {
		t.Fatal("expected 1 job")
	}

	fmt.Printf("%#v\n", jobs[0])

	if err := q.Nack(context.Background(), &taskqueue.Job{
		ID:            "test-inline-job-1",
		QueueName:     "test_inline_queue",
		StartedAt:     now,
		UpdatedAt:     now,
		Attempts:      5,
		FailureReason: "something bad happened",
		Status:        taskqueue.JobStatusCompleted,
		ProcessedBy:   "test-worker-4",
	}, &taskqueue.NackOptions{QueueName: "test_inline_queue", MaxAttemptsExceeded: true}); err != nil {
		t.Fatal(err)
	}
}
