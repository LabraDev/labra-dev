package handlers

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"labra-backend/internal/api/store"
)

func executeDeploymentAttempt(ctx context.Context, dep store.Deployment, app store.App, attempt int64) (string, error) {
	switch normalizedTrigger(dep.TriggerType) {
	case "rollback":
		return executeRollbackAttempt(ctx, dep, app)
	default:
		return executeStandardAttempt(ctx, dep.ID, app, attempt)
	}
}

func executeStandardAttempt(ctx context.Context, deploymentID int64, app store.App, attempt int64) (string, error) {
	_ = appStore.CreateDeploymentLog(ctx, deploymentID, "info", "clone repository")
	_ = appStore.CreateDeploymentLog(ctx, deploymentID, "info", fmt.Sprintf("checkout branch %s", app.Branch))

	envVars, envErr := appStore.ListAppEnvVarsForApp(ctx, app.ID)
	if envErr != nil {
		_ = appStore.CreateDeploymentLog(ctx, deploymentID, "warn", "unable to load app env vars; continuing without injected env vars")
		envVars = nil
	}
	deploymentEnv := buildDeploymentEnv(envVars)
	_ = deploymentEnv // placeholder until runner integration
	_ = appStore.CreateDeploymentLog(ctx, deploymentID, "info", describeEnvInjection(envVars))

	if forcedTransientFailures(envVars) >= attempt {
		return "", deploymentFailure{
			Message:   "simulated transient deployment error",
			Category:  "transient",
			Retryable: true,
		}
	}
	if hasEnvFlag(envVars, "LABRA_FORCE_PERMANENT_FAILURE") {
		return "", deploymentFailure{
			Message:   "simulated permanent deployment error",
			Category:  "build",
			Retryable: false,
		}
	}

	_ = appStore.CreateDeploymentLog(ctx, deploymentID, "info", "install dependencies")
	_ = appStore.CreateDeploymentLog(ctx, deploymentID, "info", "run build")
	_ = appStore.CreateDeploymentLog(ctx, deploymentID, "info", "upload static artifacts")
	_ = appStore.CreateDeploymentLog(ctx, deploymentID, "info", "invalidate CDN cache")

	if app.BuildType != "static" {
		return "", deploymentFailure{
			Message:   "unsupported build type",
			Category:  "configuration",
			Retryable: false,
		}
	}

	siteURL := strings.TrimSpace(app.SiteURL)
	if siteURL == "" {
		siteURL = fmt.Sprintf("https://%s.preview.labra.local", slugify(app.Name))
	}

	finish := store.UnixNow()
	release, releaseErr := appStore.CreateReleaseVersion(ctx, store.CreateReleaseVersionInput{
		AppID:            app.ID,
		DeploymentID:     deploymentID,
		ArtifactPath:     buildArtifactPath(app.ID, deploymentID, finish),
		ArtifactChecksum: fmt.Sprintf("sim-%d-%d", app.ID, deploymentID),
	})
	if releaseErr != nil {
		_ = appStore.CreateDeploymentLog(ctx, deploymentID, "warn", "release snapshot metadata failed to persist")
	} else {
		_ = appStore.CreateDeploymentLog(ctx, deploymentID, "info", fmt.Sprintf("published release v%d", release.VersionNumber))
		if err := appStore.ApplyReleaseRetentionPolicy(ctx, app.ID, releaseRetentionLimit(), release.ID); err != nil {
			_ = appStore.CreateDeploymentLog(ctx, deploymentID, "warn", "release retention policy failed to apply")
		}
	}

	_ = appStore.CreateDeploymentLog(ctx, deploymentID, "info", "deployment completed successfully")
	return siteURL, nil
}

func executeRollbackAttempt(ctx context.Context, dep store.Deployment, app store.App) (string, error) {
	payload, err := appStore.GetDeploymentRollbackPayload(ctx, dep.ID)
	if err != nil {
		return "", deploymentFailure{
			Message:   "rollback payload missing",
			Category:  "data",
			Retryable: false,
		}
	}

	targetRelease, err := appStore.GetReleaseVersionByIDForUser(ctx, app.ID, payload.TargetReleaseID, app.UserID)
	if err != nil {
		return "", deploymentFailure{
			Message:   "target release not found",
			Category:  "data",
			Retryable: false,
		}
	}

	_ = appStore.CreateDeploymentLog(ctx, dep.ID, "info", fmt.Sprintf("loading release v%d artifact", targetRelease.VersionNumber))
	_ = appStore.CreateDeploymentLog(ctx, dep.ID, "info", fmt.Sprintf("switch current release pointer -> %d", targetRelease.ID))

	if err := appStore.SetCurrentReleaseVersionForAppForUser(ctx, app.ID, targetRelease.ID, app.UserID); err != nil {
		return "", deploymentFailure{
			Message:   "unable to switch release pointer",
			Category:  "internal",
			Retryable: true,
		}
	}

	if err := appStore.AttachReleaseToDeployment(ctx, dep.ID, targetRelease.ID); err != nil {
		return "", deploymentFailure{
			Message:   "unable to attach target release to rollback deployment",
			Category:  "internal",
			Retryable: true,
		}
	}

	if _, err := appStore.CreateRollbackEvent(ctx, store.CreateRollbackEventInput{
		AppID:         app.ID,
		UserID:        app.UserID,
		FromReleaseID: payload.FromReleaseID,
		ToReleaseID:   targetRelease.ID,
		DeploymentID:  dep.ID,
		Reason:        payload.Reason,
	}); err != nil && !strings.Contains(strings.ToLower(err.Error()), "unique") {
		return "", deploymentFailure{
			Message:   "unable to persist rollback record",
			Category:  "internal",
			Retryable: true,
		}
	}

	targetURL := strings.TrimSpace(app.SiteURL)
	if targetURL == "" {
		targetDep, err := appStore.GetDeploymentByIDForUser(ctx, targetRelease.DeploymentID, app.UserID)
		if err == nil {
			targetURL = strings.TrimSpace(targetDep.SiteURL)
		}
	}
	if targetURL == "" {
		targetURL = fmt.Sprintf("https://%s.preview.labra.local", slugify(app.Name))
	}

	_ = appStore.CreateDeploymentLog(ctx, dep.ID, "info", fmt.Sprintf("rollback complete: now serving release v%d", targetRelease.VersionNumber))
	return targetURL, nil
}

func classifyDeploymentFailure(err error) (category string, retryable bool, reason string) {
	var f deploymentFailure
	if errors.As(err, &f) {
		return strings.TrimSpace(f.Category), f.Retryable, strings.TrimSpace(f.Message)
	}

	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		msg = "deployment failed"
	}
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "timeout") || strings.Contains(lower, "temporar") || strings.Contains(lower, "database is locked") {
		return "transient", true, msg
	}
	return "internal", false, msg
}

func retryDelaySeconds(attempt int64, cfg deploymentQueueConfig) int64 {
	if attempt <= 0 {
		attempt = 1
	}
	delay := cfg.RetryBaseSeconds
	for i := int64(1); i < attempt; i++ {
		delay *= 2
		if delay >= cfg.RetryMaxSeconds {
			return cfg.RetryMaxSeconds
		}
	}
	if delay <= 0 {
		return cfg.RetryBaseSeconds
	}
	if delay > cfg.RetryMaxSeconds {
		return cfg.RetryMaxSeconds
	}
	return delay
}

func normalizedTrigger(v string) string {
	n := strings.TrimSpace(strings.ToLower(v))
	if n == "" {
		return "manual"
	}
	return n
}

func hasEnvFlag(envVars []store.AppEnvVar, key string) bool {
	want := strings.TrimSpace(strings.ToUpper(key))
	for _, envVar := range envVars {
		if strings.TrimSpace(strings.ToUpper(envVar.Key)) != want {
			continue
		}
		value := strings.TrimSpace(strings.ToLower(envVar.Value))
		return value == "1" || value == "true" || value == "yes"
	}
	return false
}

func forcedTransientFailures(envVars []store.AppEnvVar) int64 {
	for _, envVar := range envVars {
		if strings.TrimSpace(strings.ToUpper(envVar.Key)) != "LABRA_FORCE_TRANSIENT_FAILURES" {
			continue
		}
		v, err := strconv.ParseInt(strings.TrimSpace(envVar.Value), 10, 64)
		if err != nil || v < 0 {
			return 0
		}
		if v > 10 {
			return 10
		}
		return v
	}
	return 0
}
