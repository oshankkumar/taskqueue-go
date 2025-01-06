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

	redisjs "github.com/oshankkumar/taskqueue-go/jobstore/redis"
	redisq "github.com/oshankkumar/taskqueue-go/queue/redis"
)

const ns = "taskqueue"

func main() {
	rc := redis.NewClient(&redis.Options{Addr: ":7379"})
	q := redisq.NewQueue(rc, ns)
	store := redisjs.NewStore(rc, ns)

	// Initialize Redis-backed enqueuer
	enqueuer := &taskqueue.TaskEnqueuer{
		Enqueuer: q,
		JobStore: store,
	}

	job := taskqueue.NewJob()
	err := job.MarshalPayload(map[string]string{
		"to":      "user@example.com",
		"subject": "Welcome!",
	})
	if err != nil {
		panic(err)
	}

	err = enq.Enqueue(context.Background(), job, &taskqueue.EnqueueOptions{
		QueueName: "email-jobs",
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
	"github.com/oshankkumar/taskqueue-go"
	"github.com/redis/go-redis/v9"
	"os"
	"os/signal"

	redisjs "github.com/oshankkumar/taskqueue-go/jobstore/redis"
	redisq "github.com/oshankkumar/taskqueue-go/queue/redis"
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
- Web dashboard for job management.

---

---

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

