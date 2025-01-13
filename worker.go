package taskqueue

import (
	"context"
	"errors"
	"log"
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

type InternalLogger interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
}

type printLogger struct {
	log *log.Logger
}

func (p printLogger) Debugf(format string, args ...interface{}) {
	p.log.Printf(format, args...)
}

func (p printLogger) Infof(format string, args ...interface{}) {
	p.log.Printf(format, args...)
}

type NoOpInternalLogger struct{}

func (n NoOpInternalLogger) Debugf(format string, args ...interface{}) {}
func (n NoOpInternalLogger) Infof(format string, args ...interface{})  {}

type WorkerOptions struct {
	ID             string
	Queue          Queue
	JobStore       JobStore
	ErrorHandler   func(err error)
	InternalLogger InternalLogger
}

func NewWorker(opts *WorkerOptions) *Worker {
	if opts.InternalLogger == nil {
		opts.InternalLogger = printLogger{
			log: log.New(os.Stderr, "[taskqueue] ", log.LstdFlags),
		}
	}

	if opts.ErrorHandler == nil {
		opts.ErrorHandler = func(err error) { opts.InternalLogger.Infof("failed processing job: error:%s", err) }
	}

	if opts.ID == "" {
		opts.ID, _ = os.Hostname()
	}

	return &Worker{
		ID:             opts.ID,
		Queue:          opts.Queue,
		JobStore:       opts.JobStore,
		ErrorHandler:   opts.ErrorHandler,
		InternalLogger: opts.InternalLogger,
		handlers:       make(map[string]*queueHandler),
	}
}

type Worker struct {
	ID             string
	Queue          Queue
	JobStore       JobStore
	ErrorHandler   func(err error)
	InternalLogger InternalLogger

	handlers       map[string]*queueHandler
	cancel         context.CancelFunc
	queueWaitGroup sync.WaitGroup
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
	ctx, w.cancel = context.WithCancel(ctx)
	for _, h := range w.handlers {
		go w.start(ctx, h)
	}
}

func (w *Worker) start(ctx context.Context, h *queueHandler) {
	jobCh := make(chan *Job, h.jobOptions.Concurrency)

	startWorker := func(id int) {
		w.InternalLogger.Infof("started worker %d to proceesing queue %s", id, h.queueName)
		defer w.queueWaitGroup.Done()
		for job := range jobCh {
			if err := w.processJob(ctx, job, h); err != nil {
				w.ErrorHandler(err)
			}
		}
		w.InternalLogger.Infof("stopped worker %d to processing queue %s ", id, h.queueName)
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
			w.InternalLogger.Debugf("context cancelled. stopping dequeue: %s", h.queueName)
			return
		case result := <-dequeueDone:
			w.InternalLogger.Debugf("dequeue done. result=%#v", result)

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
			w.InternalLogger.Debugf("starting dequeue from %s", h.queueName)

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

	job.FailureReason = h.handler.Handle(ctx, job)
	if job.FailureReason != nil {
		w.ErrorHandler(job.FailureReason)
	}

	job.UpdatedAt = time.Now()
	job.Attempts++

	if err := w.JobStore.CreateOrUpdate(ctx, job); err != nil {
		return err
	}

	switch {
	case job.FailureReason == nil:
		job.Status = JobStatusCompleted
	case job.Attempts >= h.jobOptions.MaxAttempts:
		job.Status = JobStatusDead
	default:
		job.Status = JobStatusFailed
	}

	if err := w.JobStore.UpdateJobStatus(ctx, job.ID, job.Status); err != nil {
		return err
	}

	if job.FailureReason == nil {
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
	w.InternalLogger.Infof("stopping worker")
	w.cancel()
	w.queueWaitGroup.Wait()
	w.InternalLogger.Infof("stopped worker")
}
