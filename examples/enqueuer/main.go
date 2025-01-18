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
	rc := redis.NewClient(&redis.Options{Addr: ":6379"})

	enq := taskqueue.NewEnqueuer(
		redisq.NewQueue(rc, redisq.WithNamespace(ns)),
		redisq.NewStore(rc, redisq.WithNamespace(ns)),
	)

	n1 := queuePaymentJob(enq)
	n2 := queueEmailJob(enq)
	n3 := queueNotificationJob(enq)

	fmt.Println("Jobs Enqueued", "payment", n1, "email", n2, "notification", n3)
}

func queueNotificationJob(enq *taskqueue.TaskEnqueuer) int {
	count := rand.Intn(100) + 100

	for range count {
		notifyJob := taskqueue.NewJob()
		_ = notifyJob.JSONMarshalPayload(map[string]interface{}{
			"to": "YOUR_DEVICE_TOKEN",
			"notification": map[string]string{
				"title": "New Message!",
				"body":  "You have a new message from John Doe.",
				"sound": "default",
			},
			"data": map[string]string{
				"message_id": "12345",
				"user_id":    "67890",
				"type":       "chat",
			},
		})
		if err := enq.Enqueue(context.Background(), notifyJob, &taskqueue.EnqueueOptions{
			QueueName: "push_notification_queue",
		}); err != nil {
			log.Fatal(err)
		}
	}

	return count
}

func queuePaymentJob(enq *taskqueue.TaskEnqueuer) int {
	count := rand.Intn(100) + 100

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
	count := rand.Intn(100) + 100

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
