# TaskQueue-Go

**TaskQueue-Go** is a high-performance, distributed task queue library for Go, designed to simplify background job processing. With support for multiple queue backends and job storage backends, along with a pluggable architecture, it provides a scalable and reliable system for decoupling task execution from your main application logic. The decoupled design enables independent scaling and optimization of the queuing system and job storage.

---

## Features

- **Distributed Task Queues**: Seamlessly enqueue and process tasks across distributed systems.
- **Customizable Queues**: Configure worker concurrency, job timeouts, and task handlers for each queue.
- **Backend Flexibility**: Initial support for Redis as a queue backend, with room for additional implementations.
- **Job Storage**: Separate and extensible storage for job metadata, with Redis and other database integrations.
- **Atomic Dequeueing**: Ensures tasks are processed reliably using Redis Lua scripts.
- **Pluggable Architecture**: Easily extend with custom implementations for enqueuing and job storage. This decoupled architecture allows you to independently scale and optimize the queuing system and job storage based on your needs.

---

## Installation

```bash
go get github.com/oshankkumar/taskqueue-go
```

---

## Getting Started

### 1. Setting Up the Enqueuer

```go
package main

import (
	"context"

	"github.com/oshankkumar/taskqueue-go"
	"github.com/redis/go-redis/v9"
	redisq "github.com/oshankkumar/taskqueue-go/redis"
)

const ns = "taskqueue"

func main() {
	rc := redis.NewClient(&redis.Options{Addr: ":7379"})
	
	// Initialize Redis-backed enqueuer
	enq := taskqueue.NewEnqueuer(
		redisq.NewQueue(rc, ns),
		redisq.NewStore(rc, ns),
	)

	job := taskqueue.NewJob()
	err := job.JSONMarshalPayload(map[string]string{
		"to":      "user@example.com",
		"subject": "Welcome!",
	})
	if err != nil {
		panic(err)
	}

	err = enq.Enqueue(context.Background(), job, &taskqueue.EnqueueOptions{
		QueueName: "email_jobs",
    })

	if err != nil {
		panic(err)
	}
}

```

### 2. Setting Up a Worker to Process Jobs

```go
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/oshankkumar/taskqueue-go"
	"github.com/redis/go-redis/v9"
	redisq "github.com/oshankkumar/taskqueue-go/redis"
)

const ns = "taskqueue"

func main() {
	rc := redis.NewClient(&redis.Options{Addr: ":7379"})
	
	worker := taskqueue.NewWorker(&taskqueue.WorkerOptions{
		Queue:       redisq.NewQueue(rc, ns),
		JobStore:    redisq.NewStore(rc, ns),
		HeartBeater: redisq.NewHeartBeater(rc, ns),
	})

	worker.RegisterHandler("email_jobs", taskqueue.HandlerFunc(func(ctx context.Context, job *taskqueue.Job) error {
		fmt.Printf("Processing job: %+v\n", job)
		return nil // Return an error if the job fails
	}), taskqueue.WithConcurrency(8))

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	worker.Start(ctx)

	<-ctx.Done()
	worker.Stop()
}

```

---

## Advanced Usage

### Custom Job Storage

You can implement your own job storage by conforming to the `JobStore` interface:

```go
type JobStore interface {
    CreateOrUpdate(ctx context.Context, job *Job) error
    GetJob(ctx context.Context, jobID string) (*Job, error)
    DeleteJob(ctx context.Context, jobID string) error
    UpdateJobStatus(ctx context.Context, jobID string, status JobStatus) error
}
```

### Redis Lua Script for Atomic Dequeueing

The library leverages a Lua script to ensure atomic dequeuing and visibility timeout management:

---

## Roadmap

- Support for additional queueing backends. (e.g., RabbitMQ, Kafka).
- Support for additional job store backends. (e.g., Mysql, Postgres).
- Metrics and monitoring integrations.
- Enhanced retry policies and dead letter queues.

---

---

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

