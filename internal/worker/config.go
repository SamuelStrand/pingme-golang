package worker

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	WorkerCount int
	BatchSize   int
	QueueSize   int
	Tick        time.Duration
}

func LoadConfigFromEnv() Config {
	workerCount := intFromEnv("WORKER_COUNT", 4)
	batchSize := intFromEnv("WORKER_BATCH_SIZE", workerCount)
	queueSize := intFromEnv("WORKER_QUEUE_SIZE", workerCount*2)

	tick := 1 * time.Second
	if rawTick := os.Getenv("SCHEDULER_TICK"); rawTick != "" {
		if parsedTick, err := time.ParseDuration(rawTick); err == nil && parsedTick > 0 {
			tick = parsedTick
		}
	}

	if workerCount < 1 {
		workerCount = 1
	}
	if batchSize < 1 {
		batchSize = workerCount
	}
	if queueSize < workerCount {
		queueSize = workerCount
	}

	return Config{
		WorkerCount: workerCount,
		BatchSize:   batchSize,
		QueueSize:   queueSize,
		Tick:        tick,
	}
}

func intFromEnv(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed < 1 {
		return fallback
	}

	return parsed
}
