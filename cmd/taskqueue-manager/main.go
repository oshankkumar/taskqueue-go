package main

import (
	"context"
	"flag"
	"github.com/oshankkumar/taskqueue-go"
	redisq "github.com/oshankkumar/taskqueue-go/redis"
	"github.com/oshankkumar/taskqueue-go/taskmanager"
	"log"
	"os/signal"
	"syscall"

	"github.com/redis/go-redis/v9"
)

var (
	namespace            = flag.String("namespace", taskqueue.DefaultNameSpace, "namespace to use")
	redisJobStoreAddr    = flag.String("redis-job-store-addr", "127.0.0.1:7379", "address of redis job store")
	redisQueueAddr       = flag.String("redis-queue-addr", "127.0.0.1:7379", "address of redis queue")
	redisHeartbeaterAddr = flag.String("redis-heartbeat-addr", "127.0.0.1:7379", "address of redis heartbeat store")
	listenAddr           = flag.String("listen", ":8050", "address to listen on")
	staticWebDir         = flag.String("static-web-dir", "./taskmanager/taskqueue-web/dist/spa", "directory to serve static files from")
)

func main() {
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	queueClient := redis.NewClient(&redis.Options{Addr: *redisQueueAddr})
	if err := queueClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("failed to connect to redis queue: %v", err)
	}

	jobClient := redis.NewClient(&redis.Options{Addr: *redisJobStoreAddr})
	if err := jobClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("failed to connect to redis job store: %v", err)
	}

	hbClient := redis.NewClient(&redis.Options{Addr: *redisHeartbeaterAddr})
	if err := hbClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("failed to connect to redis job store: %v", err)
	}

	server := taskmanager.NewServer(&taskmanager.ServerOptions{
		TaskQueue:    redisq.NewQueue(queueClient, redisq.WithNamespace(*namespace)),
		JobStore:     redisq.NewStore(jobClient, redisq.WithNamespace(*namespace)),
		HeartBeater:  redisq.NewHeartBeater(hbClient, redisq.WithNamespace(*namespace)),
		Addr:         *listenAddr,
		WebStaticDir: *staticWebDir,
	})

	if err := server.Start(ctx); err != nil {
		log.Fatal(err)
	}
}
