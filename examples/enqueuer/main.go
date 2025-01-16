package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/oshankkumar/taskqueue-go"
	redisq "github.com/oshankkumar/taskqueue-go/redis"

	"github.com/redis/go-redis/v9"
)

const ns = "taskqueue"

func main() {
	rc := redis.NewClient(&redis.Options{Addr: ":7379"})

	enq := taskqueue.NewEnqueuer(
		redisq.NewQueue(rc, ns),
		redisq.NewStore(rc, ns),
	)

	n1 := queuePaymentJob(enq)
	n2 := queueEmailJob(enq)

	fmt.Println("Jobs Enqueued", "payment", n1, "email", n2)
}

func queuePaymentJob(enq *taskqueue.TaskEnqueuer) int {
	count := rand.Intn(100)

	for i := range count {
		paymentJob := taskqueue.NewJob()
		_ = paymentJob.JSONMarshalPayload(map[string]interface{}{
			"gateway":   "razorpay",
			"amount":    500 + i,
			"wallet_id": "1",
		})
		if err := enq.Enqueue(context.Background(), paymentJob, &taskqueue.EnqueueOptions{
			QueueName: "payment_queue",
		}); err != nil {
			log.Fatal(err)
		}
	}

	return count
}

func queueEmailJob(enq *taskqueue.TaskEnqueuer) int {
	count := rand.Intn(100)

	for range count {
		job := taskqueue.NewJob()
		err := job.JSONMarshalPayload(struct {
			Sender  string    `json:"sender"`
			Message string    `json:"message"`
			SendAt  time.Time `json:"send_at"`
		}{
			Sender:  "Oshank",
			Message: "Hello World!",
			SendAt:  time.Now().Add(time.Duration(rand.Intn(1000)) * time.Hour),
		})
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Enqueuing Job:", job.ID)

		if err := enq.Enqueue(context.Background(), job, &taskqueue.EnqueueOptions{
			QueueName: "email_queue",
		}); err != nil {
			log.Fatal(err)
		}
	}

	return count
}
