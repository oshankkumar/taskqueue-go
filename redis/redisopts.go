package redis

import "time"

type Options struct {
	namespace       string
	completedJobTTL time.Duration
}

type OptFunc func(*Options)

func WithNamespace(namespace string) OptFunc {
	return func(o *Options) {
		o.namespace = namespace
	}
}

func WithCompletedJobTTL(completedJobTTL time.Duration) OptFunc {
	return func(o *Options) {
		o.completedJobTTL = completedJobTTL
	}
}
