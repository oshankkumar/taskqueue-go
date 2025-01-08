package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/oshankkumar/taskqueue-go"
	redisjs "github.com/oshankkumar/taskqueue-go/jobstore/redis"
	redisq "github.com/oshankkumar/taskqueue-go/queue/redis"

	"github.com/redis/go-redis/v9"
)

const ns = "taskqueue"

func main() {
	rc := redis.NewClient(&redis.Options{Addr: ":7379"})
	q := redisq.NewQueue(rc, ns)
	store := redisjs.NewStore(rc, ns)

	enq := &taskqueue.TaskEnqueuer{
		Enqueuer: q,
		JobStore: store,
	}

	job := taskqueue.NewJob()
	err := job.JSONMarshalPayload(struct {
		Sender  string    `json:"sender"`
		Message string    `json:"message"`
		SendAt  time.Time `json:"send_at"`
	}{
		Sender:  "Oshank",
		Message: "Hello World!",
		SendAt:  time.Now(),
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Enqueuing Job:", job.ID)

	if err := enq.Enqueue(context.Background(), job, &taskqueue.EnqueueOptions{
		QueueName: "msg_queue",
	}); err != nil {
		log.Fatal(err)
	}
}
