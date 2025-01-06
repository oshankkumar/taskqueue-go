package redis

import (
	"errors"
	"github.com/oshankkumar/taskqueue-go"
	"time"
)

type JobError struct {
	Err error
}

func (j *JobError) ScanRedis(s string) error {
	if len(s) == 0 {
		j.Err = nil
		return nil
	}

	j.Err = errors.New(s)
	return nil
}

func (j JobError) MarshalBinary() (data []byte, err error) {
	if j.Err != nil {
		return []byte(j.Err.Error()), nil
	}
	return nil, nil
}

type JobStatus int8

func (j *JobStatus) ScanRedis(s string) error {
	st, err := taskqueue.ParseJobStatus(s)
	if err != nil {
		return err
	}

	*j = JobStatus(st)
	return nil
}

func (j JobStatus) MarshalBinary() (data []byte, err error) {
	return []byte(taskqueue.JobStatus(j).String()), nil
}

type Job struct {
	ID            string    `redis:"id"`
	QueueName     string    `redis:"queue_name"`
	Payload       []byte    `redis:"payload"`
	CreatedAt     time.Time `redis:"created_at"`
	StartedAt     time.Time `redis:"started_at"`
	UpdatedAt     time.Time `redis:"updated_at"`
	Attempts      int       `redis:"attempts"`
	FailureReason JobError  `redis:"failure_reason"`
	Status        JobStatus `redis:"status"`
	ProcessedBy   string    `redis:"processed_by"`
}
