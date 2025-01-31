package redis

import (
	"context"
	"crypto/rand"
	"os"
	"testing"
	"time"

	"github.com/oshankkumar/taskqueue-go"

	"github.com/google/go-cmp/cmp"
	"github.com/redis/go-redis/v9"
)

func TestStoreCreateOrUpdate(t *testing.T) {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		t.Skip("skipping test since REDIS_ADDR is not set")
	}

	client := redis.NewClient(&redis.Options{Addr: redisAddr})
	store := NewStore(client, WithNamespace("test"))

	var payload [32]byte
	if _, err := rand.Read(payload[:]); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2025, 1, 1, 1, 10, 0, 0, time.UTC)

	job := &taskqueue.Job{
		ID:            "test-id1",
		QueueName:     "test_queue",
		Payload:       payload[:],
		CreatedAt:     now,
		StartedAt:     now,
		UpdatedAt:     now,
		Attempts:      2,
		FailureReason: taskqueue.ErrJobNotFound.Error(),
		Status:        taskqueue.JobStatusActive,
		ProcessedBy:   "test-worker-1",
	}

	if err := store.CreateOrUpdate(context.Background(), job); err != nil {
		t.Fatal(err)
	}

	got, err := store.GetJob(context.Background(), job.ID)
	if err != nil {
		t.Fatal(err)
	}

	equateErrorMessage := cmp.Comparer(func(x, y error) bool {
		if x == nil || y == nil {
			return x == nil && y == nil
		}
		return x.Error() == y.Error()
	})

	if !cmp.Equal(job, got, equateErrorMessage) {
		t.Errorf("job does not match the expected one. Diff:\n%s", cmp.Diff(job, got))
	}

	if err := store.UpdateJobStatus(context.Background(), job.ID, taskqueue.JobStatusCompleted); err != nil {
		t.Fatal(err)
	}

	got, err = store.GetJob(context.Background(), job.ID)
	if err != nil {
		t.Fatal(err)
	}

	if got.Status != taskqueue.JobStatusCompleted {
		t.Errorf("job status does not match the expected one. Diff:\n%s", cmp.Diff(job, got))
	}

	t.Log("Job updated", got.UpdatedAt)
}

func TestStoreLastHeartbeats(t *testing.T) {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		t.Skip("skipping test since REDIS_ADDR is not set")
	}

	client := redis.NewClient(&redis.Options{Addr: redisAddr})
	store := NewHeartBeater(client, WithNamespace("test"))

	hb := taskqueue.HeartbeatData{
		WorkerID:    "12345",
		StartedAt:   time.Now().UTC(),
		HeartbeatAt: time.Now().UTC(),
		Queues: []taskqueue.HeartbeatQueueData{
			{
				Name:        "queue_1",
				Concurrency: 10,
			},
			{
				Name:        "queue_2",
				Concurrency: 10,
			},
			{
				Name:        "queue_2",
				Concurrency: 10,
			},
		},
		PID: 12,
	}
	if err := store.SendHeartbeat(context.Background(), hb); err != nil {
		t.Fatal(err)
	}

	hbs, err := store.LastHeartbeats(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if !cmp.Equal(hbs[0], hb) {
		t.Fatal(cmp.Diff(hbs[0], hb))
	}
}

func TestMetricsBackend(t *testing.T) {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		t.Skip("skipping test since REDIS_ADDR is not set")
	}

	client := redis.NewClient(&redis.Options{Addr: redisAddr})

	mb := NewMetricsBackend(client, WithNamespace("test"))
	now := time.Now()

	if err := mb.RecordGauge(context.Background(), taskqueue.Metric{
		Name:   taskqueue.MetricPendingQueueSize,
		Labels: map[string]string{"name": "email_queue"},
	}, 45, now.Add(-time.Minute*120)); err != nil {
		t.Fatal(err)
	}

	if err := mb.RecordGauge(context.Background(), taskqueue.Metric{
		Name:   taskqueue.MetricPendingQueueSize,
		Labels: map[string]string{"name": "email_queue"},
	}, 60, now.Add(-time.Minute*60)); err != nil {
		t.Fatal(err)
	}

	if err := mb.RecordGauge(context.Background(), taskqueue.Metric{
		Name:   taskqueue.MetricPendingQueueSize,
		Labels: map[string]string{"name": "email_queue"},
	}, 45, now.Add(-time.Minute*45)); err != nil {
		t.Fatal(err)
	}

	if err := mb.RecordGauge(context.Background(), taskqueue.Metric{
		Name:   taskqueue.MetricPendingQueueSize,
		Labels: map[string]string{"name": "email_queue"},
	}, 32, now.Add(-time.Minute*30)); err != nil {
		t.Fatal(err)
	}

	if err := mb.RecordGauge(context.Background(), taskqueue.Metric{
		Name:   taskqueue.MetricPendingQueueSize,
		Labels: map[string]string{"name": "email_queue"},
	}, 2, now.Add(-time.Minute*15)); err != nil {
		t.Fatal(err)
	}

	if err := mb.RecordGauge(context.Background(), taskqueue.Metric{
		Name:   taskqueue.MetricPendingQueueSize,
		Labels: map[string]string{"name": "email_queue"},
	}, 80, now); err != nil {
		t.Fatal(err)
	}

	gv, err := mb.QueryRangeGaugeValues(context.Background(), taskqueue.Metric{
		Name:   taskqueue.MetricPendingQueueSize,
		Labels: map[string]string{"name": "email_queue"},
	}, now.Add(-time.Minute*120), now)
	if err != nil {
		t.Fatal(err)
	}

	if len(gv.Values) != 6 {
		t.Errorf("got %d values, want 6", len(gv.Values))
	}
}
