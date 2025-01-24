package redis

import (
	"context"
	"strconv"
	"time"

	"github.com/oshankkumar/taskqueue-go"

	"github.com/redis/go-redis/v9"
)

func NewMetricsBackend(client redis.UniversalClient, opts ...OptFunc) *MetricsBackend {
	opt := &Options{
		namespace: taskqueue.DefaultNameSpace,
	}
	for _, o := range opts {
		o(opt)
	}

	return &MetricsBackend{namespace: opt.namespace, client: client}
}

type MetricsBackend struct {
	namespace string
	client    redis.UniversalClient
}

func (m *MetricsBackend) GaugeValue(ctx context.Context, metricName string, labels map[string]string) (taskqueue.GaugeValue, error) {
	key := redisKeyMetrics(m.namespace, metricName, labels)

	result, err := m.client.ZRangeArgsWithScores(ctx, redis.ZRangeArgs{
		Key:   key,
		Start: 0,
		Stop:  0,
		Rev:   true,
	}).Result()
	if err != nil {
		return taskqueue.GaugeValue{}, err
	}

	if len(result) != 1 {
		return taskqueue.GaugeValue{}, nil
	}

	z := result[0]

	ts := time.Unix(int64(z.Score), 0)
	member, _ := z.Member.(string)
	if member == "" {
		return taskqueue.GaugeValue{}, nil
	}

	val, err := strconv.ParseInt(member, 10, 64)
	if err != nil {
		return taskqueue.GaugeValue{}, err
	}

	return taskqueue.GaugeValue{
		TimeStamp: ts,
		Value:     float64(val),
	}, nil

}

func (m *MetricsBackend) RecordGauge(ctx context.Context, metricName string, value float64, labels map[string]string, ts time.Time) error {
	key := redisKeyMetrics(m.namespace, metricName, labels)
	return m.client.ZAdd(ctx, key, redis.Z{
		Score:  float64(ts.Unix()),
		Member: int64(value),
	}).Err()
}

func (m *MetricsBackend) QueryRangeGaugeValues(ctx context.Context, metricName string, labels map[string]string, start, end time.Time) (taskqueue.GaugeRangeValue, error) {
	key := redisKeyMetrics(m.namespace, metricName, labels)

	result, err := m.client.ZRangeArgsWithScores(ctx, redis.ZRangeArgs{
		Key:     key,
		Start:   float64(start.Unix()),
		Stop:    float64(end.Unix()),
		ByScore: true,
	}).Result()
	if err != nil {
		return taskqueue.GaugeRangeValue{}, err
	}

	var gaugeRange taskqueue.GaugeRangeValue

	gaugeRange.Metric.Name = metricName
	gaugeRange.Metric.Labels = labels

	for _, z := range result {
		ts := time.Unix(int64(z.Score), 0)
		member, _ := z.Member.(string)
		if member == "" {
			continue
		}

		val, err := strconv.ParseInt(member, 10, 64)
		if err != nil {
			return taskqueue.GaugeRangeValue{}, err
		}

		gaugeRange.Values = append(gaugeRange.Values, taskqueue.GaugeValue{
			TimeStamp: ts,
			Value:     float64(val),
		})
	}

	return gaugeRange, nil
}

func redisKeyMetrics(ns string, metricName string, labels map[string]string) string {
	key := ns + ":metrics:" + metricName
	for k, v := range labels {
		key += ":" + k + ":" + v
	}
	return key
}
