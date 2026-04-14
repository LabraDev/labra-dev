package handlers

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type deploymentQueueConfig struct {
	PollInterval     time.Duration
	Concurrency      int
	MaxAttempts      int64
	RetryBaseSeconds int64
	RetryMaxSeconds  int64
}

type deploymentFailure struct {
	Message   string
	Category  string
	Retryable bool
}

func (e deploymentFailure) Error() string {
	return e.Message
}

var (
	queueMu     sync.Mutex
	queueCancel context.CancelFunc
	queueDone   chan struct{}
	queueWorker string
)

func loadDeploymentQueueConfig() deploymentQueueConfig {
	return deploymentQueueConfig{
		PollInterval:     envDurationMs("DEPLOY_QUEUE_POLL_MS", 200, 25, 5000),
		Concurrency:      envInt("DEPLOY_WORKER_CONCURRENCY", 2, 1, 16),
		MaxAttempts:      int64(envInt("DEPLOY_MAX_ATTEMPTS", 3, 1, 10)),
		RetryBaseSeconds: int64(envInt("DEPLOY_RETRY_BASE_SECONDS", 2, 1, 60)),
		RetryMaxSeconds:  int64(envInt("DEPLOY_RETRY_MAX_SECONDS", 60, 5, 600)),
	}
}

func envInt(key string, defaultV, minV, maxV int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultV
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return defaultV
	}
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

func envDurationMs(key string, defaultMs, minMs, maxMs int) time.Duration {
	v := envInt(key, defaultMs, minMs, maxMs)
	return time.Duration(v) * time.Millisecond
}

func ensureQueueStore() error {
	if appStore == nil {
		return fmt.Errorf("store not initialized")
	}
	return nil
}
