package taskqueue

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"os"
	"sync"
	"time"
)

type Handler interface {
	Handle(ctx context.Context, job *Job) error
}

type HandlerFunc func(context.Context, *Job) error

func (h HandlerFunc) Handle(ctx context.Context, job *Job) error {
	return h(ctx, job)
}

type Logger interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Error(format string, args ...interface{})
}

type WorkerOptions struct {
	ID           string
	Queue        Queue
	JobStore     JobStore
	HeartBeater  HeartBeater
	ErrorHandler func(err error)
	Logger       Logger
}

func NewWorker(opts *WorkerOptions) *Worker {
	if opts.ID == "" {
		opts.ID, _ = os.Hostname()
	}

	if opts.Logger == nil {
		opts.Logger = slog.New(slog.NewJSONHandler(os.Stdout, nil).WithAttrs([]slog.Attr{
			slog.String("worker_id", opts.ID),
		}))
	}

	if opts.ErrorHandler == nil {
		opts.ErrorHandler = func(err error) { opts.Logger.Error("failed processing job", "error", err) }
	}

	return &Worker{
		ID:             opts.ID,
		Queue:          opts.Queue,
		JobStore:       opts.JobStore,
		ErrorHandler:   opts.ErrorHandler,
		InternalLogger: opts.Logger,
		heartBeater:    opts.HeartBeater,
		handlers:       make(map[string]*queueHandler),
	}
}

type Worker struct {
	ID             string
	Queue          Queue
	JobStore       JobStore
	ErrorHandler   func(err error)
	InternalLogger Logger

	heartBeater    HeartBeater
	handlers       map[string]*queueHandler
	cancel         context.CancelFunc
	queueWaitGroup sync.WaitGroup
	startedAt      time.Time
}

type JobOptions struct {
	Timeout      time.Duration
	MaxAttempts  int
	Concurrency  int
	BackoffFunc  func(attempts int) time.Duration
	IdleWaitTime time.Duration
}

type JobOption func(*JobOptions)

func WithTimeout(timeout time.Duration) JobOption {
	return func(o *JobOptions) {
		o.Timeout = timeout
	}
}

func WithMaxAttempts(maxAttempts int) JobOption {
	return func(o *JobOptions) {
		o.MaxAttempts = maxAttempts
	}
}

func WithConcurrency(concurrency int) JobOption {
	return func(o *JobOptions) {
		o.Concurrency = concurrency
	}
}

func WithBackoffFunc(f func(attempts int) time.Duration) JobOption {
	return func(o *JobOptions) {
		o.BackoffFunc = f
	}
}

func WithIdleWaitTime(idleWaitTime time.Duration) JobOption {
	return func(o *JobOptions) {
		o.IdleWaitTime = idleWaitTime
	}
}

type queueHandler struct {
	jobOptions *JobOptions
	handler    Handler
	queueName  string
}

func (w *Worker) RegisterHandler(queueName string, h Handler, opts ...JobOption) {
	jobOpts := &JobOptions{
		Timeout:      time.Second * 10,
		MaxAttempts:  4,
		Concurrency:  4,
		IdleWaitTime: time.Second * 30,
		BackoffFunc: func(attempts int) time.Duration {
			const maxBackoff = 5 * time.Minute
			b := time.Duration(math.Pow(2, float64(attempts))) * time.Second
			if b > maxBackoff {
				b = maxBackoff
			}
			return b
		},
	}

	for _, opt := range opts {
		opt(jobOpts)
	}

	w.handlers[queueName] = &queueHandler{jobOptions: jobOpts, handler: h, queueName: queueName}
}

func (w *Worker) Start(ctx context.Context) {
	w.startedAt = time.Now()

	ctx, w.cancel = context.WithCancel(ctx)

	w.queueWaitGroup.Add(1)
	go func() {
		defer w.queueWaitGroup.Done()
		w.startHeartBeat(ctx)
		w.InternalLogger.Info("stopped heartbeater")
	}()

	w.queueWaitGroup.Add(1)
	go func() {
		defer w.queueWaitGroup.Done()
		w.reapHeartbeats(ctx)
		w.InternalLogger.Info("stopped heartbeat reaper")
	}()

	for _, h := range w.handlers {
		go w.start(ctx, h)
	}
}

func (w *Worker) start(ctx context.Context, h *queueHandler) {
	jobCh := make(chan *Job, h.jobOptions.Concurrency)

	startWorker := func(id int) {
		w.InternalLogger.Info("started worker to processing queue", "goroutine", id, "queue_name", h.queueName)
		defer w.queueWaitGroup.Done()
		for job := range jobCh {
			if err := w.processJob(ctx, job, h); err != nil {
				w.ErrorHandler(err)
			}
		}
		w.InternalLogger.Info("stopped worker to processing queue", "goroutine", id, "queue_name", h.queueName)
	}

	go w.dequeueJob(ctx, jobCh, h)

	for i := 1; i <= h.jobOptions.Concurrency; i++ {
		w.queueWaitGroup.Add(1)
		go startWorker(i)
	}
}

func (w *Worker) dequeueJob(ctx context.Context, jobCh chan<- *Job, h *queueHandler) {
	defer close(jobCh)

	type dequeueResult struct {
		jobIDs []string
		err    error
	}

	var startDequeue <-chan time.Time
	var dequeueDone chan dequeueResult
	var waitTime time.Duration

	for {
		if dequeueDone == nil {
			startDequeue = time.After(waitTime)
		}

		select {
		case <-ctx.Done():
			w.InternalLogger.Info("context cancelled. stopping dequeue", "queue_name", h.queueName)
			return
		case result := <-dequeueDone:
			w.InternalLogger.Debug("dequeue done", "result", result)

			dequeueDone = nil
			waitTime = 0

			switch {
			case errors.Is(result.err, ErrQueueEmpty):
				waitTime = h.jobOptions.IdleWaitTime
			case result.err != nil:
				w.ErrorHandler(result.err)
			default:
				for _, jobID := range result.jobIDs {
					if job, err := w.JobStore.GetJob(ctx, jobID); err != nil {
						w.ErrorHandler(err)
					} else {
						jobCh <- job
					}
				}
			}
		case <-startDequeue:
			w.InternalLogger.Debug("starting dequeue", "queue_name", h.queueName)

			dequeueDone = make(chan dequeueResult, 1)
			go func() {
				ids, err := w.Queue.Dequeue(ctx, &DequeueOptions{
					QueueName:  h.queueName,
					JobTimeout: h.jobOptions.Timeout,
				}, h.jobOptions.Concurrency)
				dequeueDone <- dequeueResult{jobIDs: ids, err: err}
			}()
		}
	}
}

func (w *Worker) processJob(ctx context.Context, job *Job, h *queueHandler) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), h.jobOptions.Timeout)
	defer cancel()

	job.StartedAt = time.Now()
	job.ProcessedBy = w.ID
	job.Status = JobStatusActive
	job.UpdatedAt = time.Now()

	if err := w.JobStore.CreateOrUpdate(ctx, job); err != nil {
		return err
	}

	jobErr := h.handler.Handle(ctx, job)
	if jobErr != nil {
		job.FailureReason = jobErr.Error()
		w.ErrorHandler(jobErr)
	}

	job.UpdatedAt = time.Now()
	job.Attempts++

	if err := w.JobStore.CreateOrUpdate(ctx, job); err != nil {
		return err
	}

	switch {
	case jobErr == nil:
		job.Status = JobStatusCompleted
	case job.Attempts >= h.jobOptions.MaxAttempts:
		job.Status = JobStatusDead
	default:
		job.Status = JobStatusFailed
	}

	if err := w.JobStore.UpdateJobStatus(ctx, job.ID, job.Status); err != nil {
		return err
	}

	if jobErr == nil {
		return w.Queue.Ack(ctx, job.ID, &AckOptions{QueueName: h.queueName})
	}

	nackOpts := &NackOptions{
		QueueName:           h.queueName,
		RetryAfter:          h.jobOptions.BackoffFunc(job.Attempts),
		MaxAttemptsExceeded: job.Attempts >= h.jobOptions.MaxAttempts,
	}

	return w.Queue.Nack(ctx, job.ID, nackOpts)
}

func (w *Worker) Stop() {
	w.InternalLogger.Info("stopping worker")
	w.cancel()
	w.queueWaitGroup.Wait()
	w.InternalLogger.Info("worker stopped")
}

func (w *Worker) startHeartBeat(ctx context.Context) {
	if w.heartBeater == nil {
		return
	}

	pid := os.Getpid()

	var queues []HeartbeatQueueData
	for _, h := range w.handlers {
		queues = append(queues, HeartbeatQueueData{
			Name:        h.queueName,
			Concurrency: h.jobOptions.Concurrency,
			MaxAttempts: h.jobOptions.MaxAttempts,
			Timeout:     h.jobOptions.Timeout,
		})
	}

	w.InternalLogger.Info("starting heartbeat loop")

	if err := w.heartBeater.SendHeartbeat(ctx, HeartbeatData{
		WorkerID:    w.ID,
		StartedAt:   w.startedAt,
		HeartbeatAt: time.Now(),
		Queues:      queues,
		PID:         pid,
	}); err != nil {
		w.ErrorHandler(err)
	}

	heartBeatTicker := time.NewTicker(time.Second * 10)
	defer heartBeatTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second*5)
			defer cancel()
			if err := w.heartBeater.RemoveHeartbeat(ctx, w.ID); err != nil {
				w.ErrorHandler(err)
			}
			return
		case <-heartBeatTicker.C:
			if err := w.heartBeater.SendHeartbeat(ctx, HeartbeatData{
				WorkerID:    w.ID,
				StartedAt:   w.startedAt,
				HeartbeatAt: time.Now(),
				Queues:      queues,
				PID:         pid,
			}); err != nil {
				w.ErrorHandler(err)
			}
		}
	}
}

func (w *Worker) reapHeartbeats(ctx context.Context) {
	if w.heartBeater == nil {
		return
	}

	ticker := time.NewTicker(time.Minute * 1)
	defer ticker.Stop()

	w.InternalLogger.Info("started reaping heartbeats")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hbs, err := w.heartBeater.LastHeartbeats(ctx)
			if err != nil {
				w.ErrorHandler(err)
				continue
			}
			for _, hb := range hbs {
				if hb.HeartbeatAt.After(time.Now().Add(-time.Minute * 5)) {
					continue
				}
				if err := w.heartBeater.RemoveHeartbeat(ctx, hb.WorkerID); err != nil {
					w.ErrorHandler(err)
				}
			}
		}
	}
}

type HeartbeatQueueData struct {
	Name        string
	Concurrency int
	MaxAttempts int
	Timeout     time.Duration
}

type HeartbeatData struct {
	WorkerID    string
	StartedAt   time.Time
	HeartbeatAt time.Time
	Queues      []HeartbeatQueueData
	PID         int
}

type HeartBeater interface {
	SendHeartbeat(ctx context.Context, data HeartbeatData) error
	RemoveHeartbeat(ctx context.Context, workerID string) error
	LastHeartbeats(ctx context.Context) ([]HeartbeatData, error)
}
