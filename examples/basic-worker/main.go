package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/oshankkumar/taskqueue-go"
	redisq "github.com/oshankkumar/taskqueue-go/redis"

	"github.com/redis/go-redis/v9"
)

const ns = "taskqueue"

func main() {
	rc := redis.NewClient(&redis.Options{Addr: ":7379"})

	worker := taskqueue.NewWorker(&taskqueue.WorkerOptions{
		Queue:       redisq.NewQueue(rc, ns),
		JobStore:    redisq.NewStore(rc, ns),
		HeartBeater: redisq.NewHeartBeater(rc, ns),
	})

	worker.RegisterHandler("email_queue", taskqueue.HandlerFunc(func(ctx context.Context, job *taskqueue.Job) error {
		var payload struct {
			Sender  string    `json:"sender"`
			Message string    `json:"message"`
			SendAt  time.Time `json:"send_at"`
		}
		if err := job.JSONUnMarshalPayload(&payload); err != nil {
			return err
		}

		fmt.Printf("Processing Job %s. Please wait...\n", job.ID)
		fmt.Printf("Received Message From %s At %s: %s\n", payload.Sender, payload.SendAt, payload.Message)

		if rand.Int31n(100) <= 30 {
			return errors.New("something went wrong")
		}

		fmt.Printf("Job %s Processed Successfully!!\n", job.ID)

		return nil
	}), taskqueue.WithConcurrency(8), taskqueue.WithMaxAttempts(2))

	worker.RegisterHandler("payment_queue", taskqueue.HandlerFunc(func(ctx context.Context, job *taskqueue.Job) error {
		if rand.Int31n(100) <= 30 {
			return errors.New("something went wrong")
		}
		return nil
	}), taskqueue.WithConcurrency(8), taskqueue.WithMaxAttempts(1))

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	worker.Start(ctx)

	<-ctx.Done()
	worker.Stop()
}
