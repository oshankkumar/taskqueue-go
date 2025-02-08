package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/oshankkumar/taskqueue-go"
	redisq "github.com/oshankkumar/taskqueue-go/redis"

	"github.com/redis/go-redis/v9"
)

const ns = "taskqueue"

var (
	id        = flag.String("id", "", "worker id")
	redisAddr = flag.String("redis-addr", ":6379", "redis address")
)

func main() {
	flag.Parse()

	rc := redis.NewClient(&redis.Options{Addr: *redisAddr})

	worker := taskqueue.NewWorker(&taskqueue.WorkerOptions{
		ID:             *id,
		Queue:          redisq.NewQueue(rc, redisq.WithNamespace(ns), redisq.WithCompletedJobTTL(time.Hour)),
		HeartBeater:    redisq.NewHeartBeater(rc, redisq.WithNamespace(ns)),
		MetricsBackend: redisq.NewMetricsBackend(rc, redisq.WithNamespace(ns)),
	})

	var (
		emailProcessed   atomic.Int32
		paymentProcessed atomic.Int32
		notifyProcessed  atomic.Int32
	)

	worker.RegisterHandler("email_queue", taskqueue.HandlerFunc(func(ctx context.Context, job *taskqueue.Job) error {
		var payload struct {
			Sender  string    `json:"sender"`
			Message string    `json:"message"`
			SendAt  time.Time `json:"send_at"`
		}
		if err := job.JSONUnMarshalPayload(&payload); err != nil {
			return err
		}

		time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond)

		if rand.Intn(100) < 30 {
			return errors.New("something bad happened")
		}

		emailProcessed.Add(1)

		fmt.Printf("job processed queue_name=email_queue job_id=%s\n", job.ID)
		return nil
	}), taskqueue.WithConcurrency(8), taskqueue.WithMaxAttempts(4))

	worker.RegisterHandler("payment_queue", taskqueue.HandlerFunc(func(ctx context.Context, job *taskqueue.Job) error {
		time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond)

		if rand.Intn(100) < 30 {
			return errors.New("something bad happened")
		}

		paymentProcessed.Add(1)
		fmt.Printf("job processed queue_name=payment_queue job_id=%s\n", job.ID)
		return nil
	}), taskqueue.WithConcurrency(8), taskqueue.WithMaxAttempts(4))

	worker.RegisterHandler("push_notification_queue", taskqueue.HandlerFunc(func(ctx context.Context, job *taskqueue.Job) error {
		time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond)

		if rand.Intn(100) < 30 {
			return taskqueue.ErrSkipRetry{Err: errors.New("something bad happened"), SkipReason: "Don't want to send outdated notification"}
		}

		notifyProcessed.Add(1)
		fmt.Printf("job processed queue_name=push_notification_queue job_id=%s\n", job.ID)
		return nil
	}), taskqueue.WithConcurrency(8), taskqueue.WithMaxAttempts(4))

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	worker.Start(ctx)

	<-ctx.Done()

	worker.Stop()

	fmt.Printf("taskqueue: shutting down. job processed email=%d payment=%d notification=%d\n",
		emailProcessed.Load(), paymentProcessed.Load(), notifyProcessed.Load(),
	)
}
