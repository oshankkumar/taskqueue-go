package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
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

	worker := taskqueue.NewWorker(&taskqueue.WorkerOptions{
		Queue:    q,
		JobStore: store,
	})

	worker.RegisterHandler("msg_queue", taskqueue.HandlerFunc(func(ctx context.Context, job *taskqueue.Job) error {
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

		<-time.After(5 * time.Second)

		fmt.Printf("Job %s Processed Successfully!!\n", job.ID)

		return nil
	}), taskqueue.WithConcurrency(8))

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	worker.Start(ctx)

	<-ctx.Done()
	worker.Stop()
}
