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

func (m *MetricsBackend) IncrementCounter(ctx context.Context, mt taskqueue.Metric, count int, ts time.Time) error {
	roundedTs := ts.Truncate(time.Minute)
	roundedTsStr := roundedTs.Format(time.RFC3339)

	hashKey := redisHashKeyCounterMetrics(m.namespace, mt.Name, mt.Labels)
	zsetKey := redisZSetKeyCounterMetrics(m.namespace, mt.Name, mt.Labels)

	_, err := m.client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.HIncrBy(ctx, hashKey, roundedTsStr, int64(count))
		pipe.ZAdd(ctx, zsetKey, redis.Z{Score: float64(roundedTs.Unix()), Member: roundedTsStr})
		return nil
	})

	return err
}

func (m *MetricsBackend) QueryRangeCounterValues(ctx context.Context, mt taskqueue.Metric, start, end time.Time) (taskqueue.MetricRangeValue, error) {
	startRoundedTs, endRoundedTs := start.Truncate(time.Minute), end.Truncate(time.Minute)

	hashKey := redisHashKeyCounterMetrics(m.namespace, mt.Name, mt.Labels)
	zsetKey := redisZSetKeyCounterMetrics(m.namespace, mt.Name, mt.Labels)

	zz, err := m.client.ZRangeArgsWithScores(ctx, redis.ZRangeArgs{
		Key:     zsetKey,
		Start:   startRoundedTs.Unix(),
		Stop:    endRoundedTs.Unix(),
		ByScore: true,
	}).Result()
	if err != nil {
		return taskqueue.MetricRangeValue{}, err
	}

	result := taskqueue.MetricRangeValue{Metric: mt}

	for _, z := range zz {
		member, _ := z.Member.(string)
		if member == "" {
			continue
		}

		val, err := m.client.HGet(ctx, hashKey, member).Int()
		if err != nil {
			continue
		}

		result.Values = append(result.Values, taskqueue.MetricValue{
			TimeStamp: time.Unix(int64(z.Score), 0),
			Value:     float64(val),
		})
	}

	return result, nil
}

func (m *MetricsBackend) GaugeValue(ctx context.Context, mt taskqueue.Metric) (taskqueue.MetricValue, error) {
	metricName, labels := mt.Name, mt.Labels
	key := redisKeyGaugeMetrics(m.namespace, metricName, labels)

	result, err := m.client.ZRangeArgsWithScores(ctx, redis.ZRangeArgs{
		Key:   key,
		Start: 0,
		Stop:  0,
		Rev:   true,
	}).Result()
	if err != nil {
		return taskqueue.MetricValue{}, err
	}

	if len(result) != 1 {
		return taskqueue.MetricValue{}, nil
	}

	z := result[0]

	member, _ := z.Member.(string)
	if member == "" {
		return taskqueue.MetricValue{}, nil
	}

	val, err := strconv.ParseInt(member, 10, 64)
	if err != nil {
		return taskqueue.MetricValue{}, err
	}

	val -= int64(z.Score)

	return taskqueue.MetricValue{
		TimeStamp: time.Unix(int64(z.Score), 0),
		Value:     float64(val),
	}, nil

}

func (m *MetricsBackend) RecordGauge(ctx context.Context, mt taskqueue.Metric, value float64, ts time.Time) error {
	metricName, labels := mt.Name, mt.Labels
	key := redisKeyGaugeMetrics(m.namespace, metricName, labels)
	score := ts.Unix()

	return m.client.ZAdd(ctx, key, redis.Z{
		Score:  float64(score),
		Member: int64(value) + score,
	}).Err()
}

func (m *MetricsBackend) QueryRangeGaugeValues(ctx context.Context, mt taskqueue.Metric, start, end time.Time) (taskqueue.MetricRangeValue, error) {
	metricName, labels := mt.Name, mt.Labels
	key := redisKeyGaugeMetrics(m.namespace, metricName, labels)

	result, err := m.client.ZRangeArgsWithScores(ctx, redis.ZRangeArgs{
		Key:     key,
		Start:   float64(start.Unix()),
		Stop:    float64(end.Unix()),
		ByScore: true,
	}).Result()
	if err != nil {
		return taskqueue.MetricRangeValue{}, err
	}

	var gaugeRange taskqueue.MetricRangeValue

	gaugeRange.Metric.Name = metricName
	gaugeRange.Metric.Labels = labels

	for _, z := range result {
		member, _ := z.Member.(string)
		if member == "" {
			continue
		}

		val, err := strconv.ParseInt(member, 10, 64)
		if err != nil {
			return taskqueue.MetricRangeValue{}, err
		}
		val -= int64(z.Score)

		gaugeRange.Values = append(gaugeRange.Values, taskqueue.MetricValue{
			TimeStamp: time.Unix(int64(z.Score), 0),
			Value:     float64(val),
		})
	}

	return gaugeRange, nil
}

func redisKeyGaugeMetrics(ns string, metricName string, labels map[string]string) string {
	key := ns + ":gauge:" + metricName
	for k, v := range labels {
		key += ":" + k + ":" + v
	}
	return key
}

func redisHashKeyCounterMetrics(ns string, metricName string, labels map[string]string) string {
	key := ns + ":counter:" + metricName
	for k, v := range labels {
		key += ":" + k + ":" + v
	}
	return key + ":values"
}

func redisZSetKeyCounterMetrics(ns string, metricName string, labels map[string]string) string {
	key := ns + ":counter:" + metricName
	for k, v := range labels {
		key += ":" + k + ":" + v
	}
	return key + ":timestamps"
}
