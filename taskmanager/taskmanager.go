package taskmanager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/rs/cors"
	"golang.org/x/sync/errgroup"

	"github.com/oshankkumar/taskqueue-go"
)

type Logger interface {
	Info(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

type ServerOptions struct {
	TaskQueue      taskqueue.Queue
	HeartBeater    taskqueue.HeartBeater
	MetricsBackend taskqueue.Metrics
	Addr           string
	WebStaticDir   string
	Logger         Logger
}

func NewServer(opts *ServerOptions) *Server {
	if opts.Logger == nil {
		opts.Logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}

	return &Server{
		taskQueue:      opts.TaskQueue,
		addr:           opts.Addr,
		webStaticDir:   opts.WebStaticDir,
		logger:         opts.Logger,
		heartBeater:    opts.HeartBeater,
		metricsBackend: opts.MetricsBackend,
	}
}

type Server struct {
	taskQueue      taskqueue.Queue
	heartBeater    taskqueue.HeartBeater
	metricsBackend taskqueue.Metrics
	addr           string
	webStaticDir   string
	logger         Logger
}

func (s *Server) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := s.initHandler()
	server := &http.Server{Addr: s.addr, Handler: mux}

	s.logger.Info("starting task manager server", "addr", s.addr)

	var eg errgroup.Group

	eg.Go(func() error {
		<-ctx.Done()
		s.logger.Info("shutting down task manager server")
		return server.Shutdown(context.Background())
	})

	eg.Go(func() error {
		defer cancel()
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})

	return eg.Wait()
}

func (s *Server) initHandler() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /api/active-workers", http.HandlerFunc(s.listActiveWorkers))
	mux.Handle("GET /api/pending-queues", http.HandlerFunc(s.listPendingQueues))
	mux.Handle("GET /api/dead-queues", http.HandlerFunc(s.listDeadQueues))
	mux.Handle("GET /api/pending-queues/{queue_name}/jobs", http.HandlerFunc(s.listPendingQueueJobs))
	mux.Handle("GET /api/dead-queues/{queue_name}/jobs", http.HandlerFunc(s.listDeadQueueJobs))
	mux.Handle("POST /api/pending-queues/{queue_name}/jobs", http.HandlerFunc(s.createJob))
	mux.Handle("DELETE /api/dead-queues/{queue_name}/jobs/{job_id}", http.HandlerFunc(s.deleteDeadJob))
	mux.Handle("POST /api/dead-queues/{queue_name}/jobs/{job_id}/requeue", http.HandlerFunc(s.requeueJob))
	mux.Handle("POST /api/pending-queues/{queue_name}/toggle-status", http.HandlerFunc(s.togglePendingQueueStatus))
	mux.Handle("POST /api/dead-queues/{queue_name}/requeue-all", http.HandlerFunc(s.requeueAllDeadJobs))
	mux.Handle("DELETE /api/dead-queues/{queue_name}/delete-all", http.HandlerFunc(s.deleteAllDeadJobs))
	mux.Handle("GET /api/metrics/jobs/processed", http.HandlerFunc(s.jobProcessedMetrics))
	mux.Handle("GET /", http.FileServer(http.Dir(s.webStaticDir)))

	handler := cors.AllowAll().Handler(mux)
	handler = s.withLog(handler)

	return handler
}

func (s *Server) withLog(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		rw := &responseWriter{ResponseWriter: w}
		h.ServeHTTP(rw, r)
		s.logger.Info("http", "method", r.Method, "uri", r.RequestURI, "took", time.Since(now).String(), "status", rw.code)
	})
}

// GET /api/metrics/jobs/processed
func (s *Server) jobProcessedMetrics(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	query, err := getMetricsRangeQueryParam(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	mt, err := s.metricsBackend.QueryRangeCounterValues(
		r.Context(),
		taskqueue.Metric{Name: taskqueue.MetricJobProcessedCount},
		query.Start,
		query.End,
	)
	if err != nil {
		http.Error(w, "Failed to query pending queues "+err.Error(), http.StatusInternalServerError)
		return
	}

	mt.Values = groupMetricsRange(query, mt.Values)

	var resp MetricsRange
	resp.Metric.Name = mt.Metric.Name
	for _, v := range mt.Values {
		resp.Values = append(resp.Values, MetricValue{
			TimeStamp: v.TimeStamp.Unix(),
			Value:     v.Value,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// POST /api/pending-queues/{queue_name}/toggle-status
func (s *Server) togglePendingQueueStatus(w http.ResponseWriter, r *http.Request) {
	queueName := r.PathValue("queue_name")
	if queueName == "" {
		http.Error(w, "Queue name is empty", http.StatusBadRequest)
		return
	}

	qInfo, err := s.taskQueue.PagePendingQueue(r.Context(), queueName, taskqueue.Pagination{Page: 1, Rows: 0})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	action := func(context.Context, string) error { return nil }
	newStatus := qInfo.Status
	if qInfo.Status == taskqueue.QueueStatusRunning {
		action = s.taskQueue.PausePendingQueue
		newStatus = taskqueue.QueueStatusPaused
	}

	if qInfo.Status == taskqueue.QueueStatusPaused {
		action = s.taskQueue.ResumePendingQueue
		newStatus = taskqueue.QueueStatusRunning
	}

	if err := action(r.Context(), queueName); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(TogglePendingQueueStatusResponse{
		OldStatus: qInfo.Status.String(),
		NewStatus: newStatus.String(),
	})
}

// GET /api/active-workers
func (s *Server) listActiveWorkers(w http.ResponseWriter, r *http.Request) {
	hbs, err := s.heartBeater.LastHeartbeats(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var resp ListActiveWorkersResponse

	for _, hb := range hbs {
		var queues []QueuesConfig
		for _, q := range hb.Queues {
			queues = append(queues, QueuesConfig{
				QueueName:   q.Name,
				Concurrency: q.Concurrency,
				MaxAttempts: q.MaxAttempts,
				Timeout:     q.Timeout,
			})
		}

		resp.ActiveWorkers = append(resp.ActiveWorkers, ActiveWorker{
			WorkerID:    hb.WorkerID,
			StartedAt:   hb.StartedAt,
			HeartbeatAt: hb.HeartbeatAt,
			Queues:      queues,
			PID:         hb.PID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// POST /api/dead-queues/{queue_name}/requeue-all
func (s *Server) requeueAllDeadJobs(w http.ResponseWriter, r *http.Request) {
	queueName := r.PathValue("queue_name")
	if queueName == "" {
		http.Error(w, "Queue name is empty", http.StatusBadRequest)
		return
	}

	for {
		queueInfo, err := s.taskQueue.PageDeadQueue(r.Context(), queueName, taskqueue.Pagination{Page: 1, Rows: 100})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if len(queueInfo.Jobs) == 0 {
			break
		}

		for _, job := range queueInfo.Jobs {
			newJob := taskqueue.NewJob()
			newJob.ID = job.ID
			newJob.Payload = job.Payload

			if err := s.taskQueue.DeleteJobFromDeadQueue(r.Context(), queueName, job.ID); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if err := s.taskQueue.Enqueue(r.Context(), newJob, &taskqueue.EnqueueOptions{QueueName: queueName}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// POST /api/dead-queues/{queue_name}/jobs/{job_id}/requeue
func (s *Server) requeueJob(w http.ResponseWriter, r *http.Request) {
	queueName, jobID := r.PathValue("queue_name"), r.PathValue("job_id")
	if queueName == "" || jobID == "" {
		http.Error(w, "Queue and job ID are required", http.StatusBadRequest)
		return
	}

	var job Job
	if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	payload, _ := json.Marshal(job.Args)

	newJob := taskqueue.NewJob()
	newJob.ID = jobID
	newJob.Payload = payload

	if err := s.taskQueue.DeleteJobFromDeadQueue(r.Context(), queueName, jobID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := s.taskQueue.Enqueue(r.Context(), newJob, &taskqueue.EnqueueOptions{QueueName: queueName}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

// DELETE /api/dead-queues/{queue_name}/delete-all
func (s *Server) deleteAllDeadJobs(w http.ResponseWriter, r *http.Request) {
	queueName := r.PathValue("queue_name")
	if queueName == "" {
		http.Error(w, "Queue name is empty", http.StatusBadRequest)
		return
	}

	for {
		queueInfo, err := s.taskQueue.PageDeadQueue(r.Context(), queueName, taskqueue.Pagination{Page: 1, Rows: 100})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if len(queueInfo.Jobs) == 0 {
			break
		}

		for _, job := range queueInfo.Jobs {
			if err := s.taskQueue.DeleteJobFromDeadQueue(r.Context(), queueName, job.ID); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

	}

	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/dead-queues/{queue_name}/jobs/{job_id}
func (s *Server) deleteDeadJob(w http.ResponseWriter, r *http.Request) {
	queueName, jobID := r.PathValue("queue_name"), r.PathValue("job_id")
	if queueName == "" || jobID == "" {
		http.Error(w, "Queue and job ID are required", http.StatusBadRequest)
		return
	}

	if err := s.taskQueue.DeleteJobFromDeadQueue(r.Context(), queueName, jobID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// POST /api/pending-queues/{queue_name}/jobs
func (s *Server) createJob(w http.ResponseWriter, r *http.Request) {
	queueName := r.PathValue("queue_name")
	if queueName == "" {
		http.Error(w, "Queue name is required", http.StatusBadRequest)
		return
	}

	var createJobReq CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&createJobReq); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	job := taskqueue.NewJob()
	if err := job.JSONMarshalPayload(createJobReq.Args); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.taskQueue.Enqueue(r.Context(), job, &taskqueue.EnqueueOptions{QueueName: queueName}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(Job{
		ID:        job.ID,
		QueueName: queueName,
		Args:      createJobReq.Args,
		CreatedAt: job.CreatedAt,
		Status:    job.Status.String(),
	})
}

// GET /api/pending-queues
func (s *Server) listPendingQueues(w http.ResponseWriter, r *http.Request) {
	queues, err := s.taskQueue.ListPendingQueues(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var resp ListQueueResponse
	for _, queue := range queues {
		resp.Queues = append(resp.Queues, QueueInfo{
			NameSpace: queue.NameSpace,
			Name:      queue.Name,
			JobCount:  queue.JobCount,
			Status:    queue.Status.String(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// GET /api/dead-queues
func (s *Server) listDeadQueues(w http.ResponseWriter, r *http.Request) {
	queues, err := s.taskQueue.ListDeadQueues(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var resp ListQueueResponse
	for _, queue := range queues {
		resp.Queues = append(resp.Queues, QueueInfo{
			NameSpace: queue.NameSpace,
			Name:      queue.Name,
			JobCount:  queue.JobCount,
			Status:    queue.Status.String(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// GET /api/pending-queues/{queue_name}/jobs
func (s *Server) listPendingQueueJobs(w http.ResponseWriter, r *http.Request) {
	queueName := r.PathValue("queue_name")
	if queueName == "" {
		http.Error(w, "Queue name is required", http.StatusBadRequest)
		return
	}

	p, err := getPagination(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	queueDetails, err := s.taskQueue.PagePendingQueue(r.Context(), queueName, p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.listQueueJobs(w, r, queueDetails, p)
}

// GET /api/dead-queues/{queue_name}/jobs
func (s *Server) listDeadQueueJobs(w http.ResponseWriter, r *http.Request) {
	queueName := r.PathValue("queue_name")
	if queueName == "" {
		http.Error(w, "Queue name is required", http.StatusBadRequest)
		return
	}

	p, err := getPagination(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	queueDetails, err := s.taskQueue.PageDeadQueue(r.Context(), queueName, p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.listQueueJobs(w, r, queueDetails, p)
}

func (s *Server) listQueueJobs(w http.ResponseWriter, r *http.Request, queueDetails *taskqueue.QueueDetails, p taskqueue.Pagination) {
	resp := &ListQueueJobsResponse{
		NameSpace:  queueDetails.NameSpace,
		Name:       queueDetails.Name,
		JobCount:   queueDetails.JobCount,
		Status:     queueDetails.Status.String(),
		Pagination: p,
	}
	for _, job := range queueDetails.Jobs {
		var args interface{}
		_ = json.Unmarshal(job.Payload, &args)

		resp.Jobs = append(resp.Jobs, Job{
			ID:            job.ID,
			QueueName:     job.QueueName,
			Args:          args,
			CreatedAt:     job.CreatedAt,
			StartedAt:     job.StartedAt,
			UpdatedAt:     job.UpdatedAt,
			Attempts:      job.Attempts,
			FailureReason: job.FailureReason,
			Status:        job.Status.String(),
			ProcessedBy:   job.ProcessedBy,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func getPagination(r *http.Request) (taskqueue.Pagination, error) {
	page, count := r.URL.Query().Get("page"), r.URL.Query().Get("rows")

	if page == "" {
		page = "1"
	}

	if count == "" {
		count = "10"
	}

	pageNum, err := strconv.Atoi(page)
	if err != nil {
		return taskqueue.Pagination{}, err
	}

	countNum, err := strconv.Atoi(count)
	if err != nil {
		return taskqueue.Pagination{}, err
	}

	return taskqueue.Pagination{Page: pageNum, Rows: countNum}, nil
}

func getMetricsRangeQueryParam(r *http.Request) (MetricsQueryParam, error) {
	startTimeStr := r.URL.Query().Get("start")

	startTime := time.Now().Add(-time.Hour * 24)
	if startTimeStr != "" {
		startTimeUnix, err := strconv.ParseInt(startTimeStr, 10, 64)
		if err != nil {
			return MetricsQueryParam{}, err
		}
		startTime = time.Unix(startTimeUnix, 0)
	}

	endTimeStr := r.URL.Query().Get("end")
	endTime := time.Now()
	if endTimeStr != "" {
		endTimeUnix, err := strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			return MetricsQueryParam{}, err
		}
		endTime = time.Unix(endTimeUnix, 0)
	}

	step := r.URL.Query().Get("step")
	if step == "" {
		step = "1"
	}

	stepInt, err := strconv.Atoi(step)
	if err != nil {
		return MetricsQueryParam{}, err
	}

	fmt.Println(startTime, endTime, stepInt)

	return MetricsQueryParam{
		Start: startTime,
		End:   endTime,
		Step:  time.Duration(stepInt) * time.Second,
	}, nil
}

type responseWriter struct {
	http.ResponseWriter
	code int
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.code = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func groupMetricsRange(p MetricsQueryParam, values []taskqueue.MetricValue) []taskqueue.MetricValue {
	var result []taskqueue.MetricValue

	start, end, step := p.Start.Unix(), p.End.Unix(), int64(p.Step.Seconds())
	j := 0

	for i := start; i <= end; i += step {
		val := taskqueue.MetricValue{
			TimeStamp: time.Unix(i, 0),
		}
		for ; j < len(values) && values[j].TimeStamp.Unix() <= i; j++ {
			val.Value += values[j].Value
		}
		result = append(result, val)
	}

	return result
}
