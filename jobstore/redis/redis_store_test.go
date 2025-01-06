package redis

import (
	"context"
	"github.com/oshankkumar/taskqueue-go"
	"github.com/redis/go-redis/v9"
	"testing"
	"time"
)

func TestStore_CreateOrUpdate(t *testing.T) {
	c := redis.NewClient(&redis.Options{
		Addr: ":7379",
	})

	store := NewStore(c, "")

	payload := `{
    "guid": "25f5efc6-ddd8-4b2c-ae7a-4fc012c13bb8",
    "isActive": true,
    "balance": "$3,132.90",
    "picture": "http://placehold.it/32x32",
    "age": 39,
    "eyeColor": "brown",
    "name": "Lavonne Garner",
    "gender": "female",
    "company": "INSOURCE"
}`
	err := store.CreateOrUpdate(context.Background(), &taskqueue.Job{
		ID:            "1234567",
		QueueName:     "ns:queue:myqueue",
		Payload:       []byte(payload),
		CreatedAt:     time.Now().Add(-time.Second * 10),
		StartedAt:     time.Now(),
		UpdatedAt:     time.Now().Add(time.Second * 10),
		Attempts:      1,
		FailureReason: taskqueue.ErrQueueEmpty,
		Status:        taskqueue.JobStatusActive,
		ProcessedBy:   "w1",
	})

	if err != nil {
		t.Fatal(err)
	}

	job, err := store.GetJob(context.Background(), "1234567")
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%#v", job)
	t.Log(string(job.Payload))
}
