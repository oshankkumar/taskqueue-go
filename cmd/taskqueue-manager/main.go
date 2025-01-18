package main

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"syscall"

	"github.com/caarlos0/env/v11"
	redisq "github.com/oshankkumar/taskqueue-go/redis"
	"github.com/oshankkumar/taskqueue-go/taskmanager"
	"github.com/redis/go-redis/v9"
)

var (
	namespace            = flag.String("namespace", "", "namespace to use. [env: NAMESPACE, default: taskqueue]")
	redisJobStoreAddr    = flag.String("redis-job-store-addr", "", "address of redis job store. [env: REDIS_JOB_STORE_ADDR, default: 127.0.0.1:6379]")
	redisQueueAddr       = flag.String("redis-queue-addr", "", "address of redis queue. [env: REDIS_QUEUE_ADDR, default: 127.0.0.1:6379]")
	redisHeartbeaterAddr = flag.String("redis-heartbeat-addr", "", "address of redis heartbeat store. [env: REDIS_HEARTBEAT_ADDR, default: 127.0.0.1:6379]")
	listenAddr           = flag.String("listen", "", "address to listen. [env: LISTEN_ADDR, default: :8050]")
	webStaticDir         = flag.String("web-static-dir", "", "directory to serve static files. [env: WEB_STATIC_DIR, default: ./taskmanager/taskqueue-web/dist/spa]")
)

type config struct {
	Namespace            string `env:"NAMESPACE" envDefault:"taskqueue"`
	RedisJobStoreAddr    string `env:"REDIS_JOB_STORE_ADDR" envDefault:"127.0.0.1:6379"`
	RedisQueueAddr       string `env:"REDIS_QUEUE_ADDR" envDefault:"127.0.0.1:6379"`
	RedisHeartbeaterAddr string `env:"REDIS_HEARTBEAT_ADDR" envDefault:"127.0.0.1:6379"`
	ListenAddr           string `env:"LISTEN_ADDR" envDefault:":8050"`
	WebStaticDir         string `env:"WEB_STATIC_DIR" envDefault:"./taskmanager/taskqueue-web/dist/spa"`
}

func appConfig() config {
	flag.Parse()

	var cfg config
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed parsing configuration %+v", err)
	}

	if *namespace != "" {
		cfg.Namespace = *namespace
	}

	if *redisJobStoreAddr != "" {
		cfg.RedisJobStoreAddr = *redisJobStoreAddr
	}

	if *redisQueueAddr != "" {
		cfg.RedisQueueAddr = *redisQueueAddr
	}

	if *redisHeartbeaterAddr != "" {
		cfg.RedisHeartbeaterAddr = *redisHeartbeaterAddr
	}

	if *listenAddr != "" {
		cfg.ListenAddr = *listenAddr
	}

	if *webStaticDir != "" {
		cfg.WebStaticDir = *webStaticDir
	}

	return cfg
}

func main() {
	cfg := appConfig()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	queueClient := redis.NewClient(&redis.Options{Addr: cfg.RedisQueueAddr})
	if err := queueClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("failed to connect to redis queue: %v", err)
	}

	jobClient := redis.NewClient(&redis.Options{Addr: cfg.RedisJobStoreAddr})
	if err := jobClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("failed to connect to redis job store: %v", err)
	}

	hbClient := redis.NewClient(&redis.Options{Addr: cfg.RedisHeartbeaterAddr})
	if err := hbClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("failed to connect to redis job store: %v", err)
	}

	server := taskmanager.NewServer(&taskmanager.ServerOptions{
		TaskQueue:    redisq.NewQueue(queueClient, redisq.WithNamespace(cfg.Namespace)),
		JobStore:     redisq.NewStore(jobClient, redisq.WithNamespace(cfg.Namespace)),
		HeartBeater:  redisq.NewHeartBeater(hbClient, redisq.WithNamespace(cfg.Namespace)),
		Addr:         cfg.ListenAddr,
		WebStaticDir: cfg.WebStaticDir,
	})

	if err := server.Start(ctx); err != nil {
		log.Fatal(err)
	}
}
