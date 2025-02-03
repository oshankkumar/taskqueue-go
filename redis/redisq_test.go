package redis

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/oshankkumar/taskqueue-go"

	"github.com/redis/go-redis/v9"
)

const testPayload = `{
	"guid": "f2d64811-bb24-465f-8a89-86c2e84938bd",
    "isActive": false,
    "balance": "$3,822.26",
    "picture": "http://placehold.it/32x32",
    "age": 30,
    "eyeColor": "brown",
    "name": "Robin Hernandez",
    "gender": "female",
    "company": "CAPSCREEN",
    "email": "robinhernandez@capscreen.com",
    "phone": "+1 (823) 515-3571"
}`

func TestRedisQueue(t *testing.T) {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		t.Skip("Skipping TestRedisQueue. Please set REDIS_ADDR environment variable.")
	}

	client := redis.NewClient(&redis.Options{Addr: redisAddr})

	q := NewQueue(client, WithCompletedJobTTL(time.Minute*30))

	client.Del(context.Background(), redisKeyPendingQueue(taskqueue.DefaultNameSpace, "test_redis_queue"))

	job := taskqueue.NewJob()
	job.Payload = []byte(testPayload)

	if err := q.Enqueue(context.Background(), job, &taskqueue.EnqueueOptions{QueueName: "test_redis_queue"}); err != nil {
		t.Fatal(err)
	}

	jobs, err := q.Dequeue(context.Background(), &taskqueue.DequeueOptions{QueueName: "test_redis_queue"}, 1)
	if err != nil {
		t.Fatal(err)
	}

	if len(jobs) != 1 {
		t.Fatal("expected 1 job")
	}

	now := time.Now()
	deqJob := jobs[0]

	if !bytes.Equal(deqJob.Payload, job.Payload) {
		t.Fatal("expected payload to be equal to test payload")
	}

	if deqJob.ID != job.ID {
		t.Fatalf("expected ID to be equal to job ID, expected: %s got: %s", job.ID, deqJob.ID)
	}

	if deqJob.Status != taskqueue.JobStatusActive {
		t.Error("expected status to be active after dequeue got:", deqJob.Status)
	}

	deqJob.Status = taskqueue.JobStatusCompleted
	deqJob.UpdatedAt = now
	deqJob.Attempts = 3
	deqJob.ProcessedBy = "test-worker-0"

	if err := q.Ack(context.Background(), deqJob, &taskqueue.AckOptions{QueueName: "test_redis_queue"}); err != nil {
		t.Fatal(err)
	}

	job, err = q.getJob(context.Background(), deqJob.ID)
	if err != nil {
		t.Fatal(err)
	}

	if job.Status != taskqueue.JobStatusCompleted {
		t.Fatal("expected job to be completed")
	}

	if job.UpdatedAt.Unix() != now.Unix() {
		t.Fatalf("expected job.Updated = %s got = %s", now, job.UpdatedAt)
	}

	if job.ProcessedBy != "test-worker-0" {
		t.Fatal("expected job to be processed by test-worker-0")
	}

	if job.Attempts != 3 {
		t.Fatal("expected job to have 3 attempts")
	}

	dur, err := client.TTL(context.Background(), redisKeyJob(taskqueue.DefaultNameSpace, deqJob.ID)).Result()
	if err != nil {
		t.Fatal(err)
	}

	if dur != time.Minute*30 {
		t.Fatal("expected job to have a TTL set")
	}
}
