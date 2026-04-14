package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"labra-backend/internal/api/store"
)

func StartDeploymentQueueWorker() error {
	queueMu.Lock()
	defer queueMu.Unlock()

	if err := ensureQueueStore(); err != nil {
		return err
	}
	if queueCancel != nil {
		return nil
	}

	cfg := loadDeploymentQueueConfig()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	workerID := fmt.Sprintf("worker-%d", time.Now().UnixNano())

	queueCancel = cancel
	queueDone = done
	queueWorker = workerID

	go runDeploymentQueueLoop(ctx, done, cfg, workerID)
	return nil
}

func StopDeploymentQueueWorker() {
	queueMu.Lock()
	cancel := queueCancel
	done := queueDone
	queueCancel = nil
	queueDone = nil
	queueWorker = ""
	queueMu.Unlock()

	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

func enqueueDeploymentJob(ctx context.Context, dep store.Deployment) (store.DeploymentJob, error) {
	job, err := appStore.CreateDeploymentJob(ctx, store.CreateDeploymentJobInput{
		DeploymentID: dep.ID,
		AppID:        dep.AppID,
		UserID:       dep.UserID,
		MaxAttempts:  loadDeploymentQueueConfig().MaxAttempts,
	})
	if err == nil {
		return job, nil
	}

	if strings.Contains(strings.ToLower(err.Error()), "unique constraint failed") {
		return appStore.GetDeploymentJobByDeploymentID(ctx, dep.ID)
	}
	return store.DeploymentJob{}, err
}

func runDeploymentQueueLoop(ctx context.Context, done chan struct{}, cfg deploymentQueueConfig, workerID string) {
	defer close(done)

	sem := make(chan struct{}, cfg.Concurrency)
	sleep := time.NewTicker(cfg.PollInterval)
	defer sleep.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		job, err := appStore.ClaimNextRunnableDeploymentJob(ctx, workerID)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				select {
				case <-ctx.Done():
					return
				case <-sleep.C:
					continue
				}
			}
			select {
			case <-ctx.Done():
				return
			case <-sleep.C:
				continue
			}
		}

		sem <- struct{}{}
		go func(job store.DeploymentJob) {
			defer func() { <-sem }()
			processDeploymentJob(ctx, job, cfg)
		}(job)
	}
}

func processDeploymentJob(ctx context.Context, job store.DeploymentJob, cfg deploymentQueueConfig) {
	dep, err := appStore.GetDeploymentByIDForUser(ctx, job.DeploymentID, job.UserID)
	if err != nil {
		_, _ = appStore.MarkDeploymentJobFailed(ctx, job.ID, "deployment not found for queued job", "data")
		return
	}

	app, err := appStore.GetAppByIDForUser(ctx, dep.AppID, dep.UserID)
	if err != nil {
		_, _ = appStore.MarkDeploymentJobFailed(ctx, job.ID, "app not found for queued job", "data")
		_, _ = appStore.UpdateDeploymentOutcome(ctx, dep.ID, "failed", "app not found for queued job", "data", false, "", dep.StartedAt, store.UnixNow())
		return
	}

	start := dep.StartedAt
	if start <= 0 {
		start = store.UnixNow()
	}
	_, _ = appStore.UpdateDeploymentOutcome(ctx, dep.ID, "running", "", "", false, dep.SiteURL, start, 0)
	_ = appStore.CreateDeploymentLog(ctx, dep.ID, "info", fmt.Sprintf("worker picked up job attempt %d/%d", job.AttemptCount, job.MaxAttempts))

	siteURL, execErr := executeDeploymentAttempt(ctx, dep, app, job.AttemptCount)
	if execErr == nil {
		finish := store.UnixNow()
		_, _ = appStore.MarkDeploymentJobSucceeded(ctx, job.ID)
		_, _ = appStore.UpdateDeploymentOutcome(ctx, dep.ID, "succeeded", "", "", false, siteURL, start, finish)
		_ = appStore.RecordAppDeploymentOutcome(ctx, dep.AppID, "succeeded", start, finish, normalizedTrigger(dep.TriggerType))
		return
	}

	category, retryable, reason := classifyDeploymentFailure(execErr)
	_ = appStore.CreateDeploymentLog(ctx, dep.ID, "error", fmt.Sprintf("attempt %d failed: %s", job.AttemptCount, reason))

	if retryable && job.AttemptCount < job.MaxAttempts {
		delay := retryDelaySeconds(job.AttemptCount, cfg)
		nextAttempt := store.UnixNow() + delay
		_, _ = appStore.MarkDeploymentJobRetry(ctx, job.ID, nextAttempt, reason, category)
		_, _ = appStore.UpdateDeploymentOutcome(ctx, dep.ID, "queued", reason, category, true, dep.SiteURL, start, 0)
		_ = appStore.CreateDeploymentLog(ctx, dep.ID, "warn", fmt.Sprintf("retry scheduled in %ds (attempt %d/%d)", delay, job.AttemptCount+1, job.MaxAttempts))
		return
	}

	finish := store.UnixNow()
	_, _ = appStore.MarkDeploymentJobFailed(ctx, job.ID, reason, category)
	_, _ = appStore.UpdateDeploymentOutcome(ctx, dep.ID, "failed", reason, category, retryable, "", start, finish)
	_ = appStore.RecordAppDeploymentOutcome(ctx, dep.AppID, "failed", start, finish, normalizedTrigger(dep.TriggerType))
}
