package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"labra-backend/internal/api/store"
)

func TestCreateDeployHandlerQueuesAndProcessesJob(t *testing.T) {
	db := setupPhase4TestDB(t)
	defer db.Close()

	app := createTestApp(t, 1, "phase8-app", "owner/repo", "main")
	deploymentID := triggerManualDeployViaHandler(t, app.ID, 1)

	waitForDeploymentTerminalStatusWithTimeout(t, deploymentID, app.UserID, 5*time.Second)

	dep, err := appStore.GetDeploymentByIDForUser(context.Background(), deploymentID, app.UserID)
	if err != nil {
		t.Fatalf("get deployment: %v", err)
	}
	if dep.Status != "succeeded" {
		t.Fatalf("expected succeeded deployment, got %q", dep.Status)
	}
	if dep.ReleaseID == 0 {
		t.Fatalf("expected release_id to be set for successful deployment")
	}

	job, err := appStore.GetDeploymentJobByDeploymentID(context.Background(), deploymentID)
	if err != nil {
		t.Fatalf("get deployment job: %v", err)
	}
	if job.Status != "succeeded" {
		t.Fatalf("expected succeeded job, got %q", job.Status)
	}
	if job.AttemptCount != 1 {
		t.Fatalf("expected attempt_count=1, got %d", job.AttemptCount)
	}
}

func TestDeploymentQueueRetriesTransientFailures(t *testing.T) {
	db := setupPhase4TestDB(t)
	defer db.Close()

	app := createTestApp(t, 1, "phase8-retry", "owner/repo", "main")
	_, err := appStore.CreateAppEnvVar(context.Background(), app.ID, app.UserID, store.CreateAppEnvVarInput{
		Key:      "LABRA_FORCE_TRANSIENT_FAILURES",
		Value:    "1",
		IsSecret: false,
	})
	if err != nil {
		t.Fatalf("create env var: %v", err)
	}

	deploymentID := triggerManualDeployViaHandler(t, app.ID, 1)
	waitForDeploymentTerminalStatusWithTimeout(t, deploymentID, app.UserID, 7*time.Second)

	dep, err := appStore.GetDeploymentByIDForUser(context.Background(), deploymentID, app.UserID)
	if err != nil {
		t.Fatalf("get deployment: %v", err)
	}
	if dep.Status != "succeeded" {
		t.Fatalf("expected deployment succeeded after retry, got %q", dep.Status)
	}

	job, err := appStore.GetDeploymentJobByDeploymentID(context.Background(), deploymentID)
	if err != nil {
		t.Fatalf("get deployment job: %v", err)
	}
	if job.AttemptCount < 2 {
		t.Fatalf("expected at least 2 attempts for transient retry, got %d", job.AttemptCount)
	}
	if job.Status != "succeeded" {
		t.Fatalf("expected final job status succeeded, got %q", job.Status)
	}

	logs, err := appStore.ListDeploymentLogs(context.Background(), deploymentID)
	if err != nil {
		t.Fatalf("list deployment logs: %v", err)
	}
	foundRetryLog := false
	for _, line := range logs {
		if strings.Contains(strings.ToLower(line.Message), "retry scheduled") {
			foundRetryLog = true
			break
		}
	}
	if !foundRetryLog {
		t.Fatalf("expected retry scheduled log, got logs=%#v", logs)
	}
}

func TestDeploymentQueueFailsWithoutRetryForConfigurationErrors(t *testing.T) {
	db := setupPhase4TestDB(t)
	defer db.Close()

	app := createTestApp(t, 1, "phase8-noretry", "owner/repo", "main")
	_, err := appStore.UpdateAppForUser(context.Background(), app.ID, app.UserID, store.UpdateAppInput{
		Name:              app.Name,
		Branch:            app.Branch,
		BuildType:         "container",
		OutputDir:         app.OutputDir,
		RootDir:           app.RootDir,
		SiteURL:           app.SiteURL,
		AutoDeployEnabled: app.AutoDeployEnabled,
	})
	if err != nil {
		t.Fatalf("update app build type: %v", err)
	}

	deploymentID := triggerManualDeployViaHandler(t, app.ID, 1)
	waitForDeploymentTerminalStatusWithTimeout(t, deploymentID, app.UserID, 5*time.Second)

	dep, err := appStore.GetDeploymentByIDForUser(context.Background(), deploymentID, app.UserID)
	if err != nil {
		t.Fatalf("get deployment: %v", err)
	}
	if dep.Status != "failed" {
		t.Fatalf("expected failed deployment, got %q", dep.Status)
	}
	if dep.FailureCategory != "configuration" {
		t.Fatalf("expected failure_category=configuration, got %q", dep.FailureCategory)
	}
	if dep.Retryable {
		t.Fatalf("expected retryable=false for configuration failure")
	}

	job, err := appStore.GetDeploymentJobByDeploymentID(context.Background(), deploymentID)
	if err != nil {
		t.Fatalf("get deployment job: %v", err)
	}
	if job.Status != "failed" {
		t.Fatalf("expected job status failed, got %q", job.Status)
	}
	if job.AttemptCount != 1 {
		t.Fatalf("expected attempt_count=1 for non-retryable failure, got %d", job.AttemptCount)
	}
}

func TestRollbackHandlerQueuesPayloadAndProcessesViaWorker(t *testing.T) {
	db := setupPhase4TestDB(t)
	defer db.Close()

	app := createTestApp(t, 1, "phase8-rollback", "owner/repo", "main")

	dep1, err := appStore.CreateDeployment(context.Background(), store.CreateDeploymentInput{
		AppID:         app.ID,
		UserID:        app.UserID,
		Status:        "queued",
		TriggerType:   "manual",
		Branch:        app.Branch,
		CorrelationID: "phase8-rb-1",
	})
	if err != nil {
		t.Fatalf("create dep1: %v", err)
	}
	runManualDeployment(dep1.ID, app)

	dep2, err := appStore.CreateDeployment(context.Background(), store.CreateDeploymentInput{
		AppID:         app.ID,
		UserID:        app.UserID,
		Status:        "queued",
		TriggerType:   "manual",
		Branch:        app.Branch,
		CorrelationID: "phase8-rb-2",
	})
	if err != nil {
		t.Fatalf("create dep2: %v", err)
	}
	runManualDeployment(dep2.ID, app)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/apps/%d/rollback", app.ID), bytes.NewBufferString(`{"reason":"phase8 rollback test"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "1")
	rr := httptest.NewRecorder()
	CreateRollbackHandler(rr, req)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusAccepted, rr.Code, rr.Body.String())
	}

	var response struct {
		Deployment struct {
			ID int64 `json:"id"`
		} `json:"deployment"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode rollback response: %v", err)
	}

	payload, err := appStore.GetDeploymentRollbackPayload(context.Background(), response.Deployment.ID)
	if err != nil {
		t.Fatalf("expected rollback payload persisted: %v", err)
	}
	if payload.TargetReleaseID == 0 {
		t.Fatalf("expected non-zero target_release_id in payload")
	}

	waitForDeploymentTerminalStatusWithTimeout(t, response.Deployment.ID, app.UserID, 7*time.Second)

	job, err := appStore.GetDeploymentJobByDeploymentID(context.Background(), response.Deployment.ID)
	if err != nil {
		t.Fatalf("get rollback job: %v", err)
	}
	if job.Status != "succeeded" {
		t.Fatalf("expected rollback job succeeded, got %q", job.Status)
	}
}

func TestEnqueueDeploymentJobIsIdempotentPerDeployment(t *testing.T) {
	db := setupPhase4TestDB(t)
	defer db.Close()

	app := createTestApp(t, 1, "phase8-idempotent", "owner/repo", "main")
	dep, err := appStore.CreateDeployment(context.Background(), store.CreateDeploymentInput{
		AppID:         app.ID,
		UserID:        app.UserID,
		Status:        "queued",
		TriggerType:   "manual",
		Branch:        app.Branch,
		CorrelationID: "phase8-idempotency",
	})
	if err != nil {
		t.Fatalf("create deployment: %v", err)
	}

	job1, err := enqueueDeploymentJob(context.Background(), dep)
	if err != nil {
		t.Fatalf("enqueue first job: %v", err)
	}
	job2, err := enqueueDeploymentJob(context.Background(), dep)
	if err != nil {
		t.Fatalf("enqueue second job: %v", err)
	}
	if job1.ID != job2.ID {
		t.Fatalf("expected idempotent queue insert to return same job id, got %d and %d", job1.ID, job2.ID)
	}
}

func TestGetDeployQueueStatusHandler(t *testing.T) {
	db := setupPhase4TestDB(t)
	defer db.Close()

	app := createTestApp(t, 1, "phase8-queue-api", "owner/repo", "main")
	deploymentID := triggerManualDeployViaHandler(t, app.ID, 1)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/deploys/%d/queue", deploymentID), nil)
	req.Header.Set("X-User-ID", "1")
	rr := httptest.NewRecorder()
	GetDeployQueueStatusHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var body struct {
		DeploymentID int64               `json:"deployment_id"`
		Job          store.DeploymentJob `json:"job"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode queue status response: %v", err)
	}
	if body.DeploymentID != deploymentID {
		t.Fatalf("expected deployment_id=%d, got %d", deploymentID, body.DeploymentID)
	}
	if body.Job.DeploymentID != deploymentID {
		t.Fatalf("expected job deployment_id=%d, got %d", deploymentID, body.Job.DeploymentID)
	}
}

func triggerManualDeployViaHandler(t *testing.T, appID, userID int64) int64 {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/apps/%d/deploy", appID), nil)
	req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
	rr := httptest.NewRecorder()
	CreateDeployHandler(rr, req)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusAccepted, rr.Code, rr.Body.String())
	}

	var response struct {
		Deployment struct {
			ID int64 `json:"id"`
		} `json:"deployment"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode create deploy response: %v", err)
	}
	if response.Deployment.ID <= 0 {
		t.Fatalf("expected valid deployment id in response: %s", rr.Body.String())
	}
	return response.Deployment.ID
}

func waitForDeploymentTerminalStatusWithTimeout(t *testing.T, deploymentID, userID int64, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		dep, err := appStore.GetDeploymentByIDForUser(context.Background(), deploymentID, userID)
		if err == nil && (dep.Status == "succeeded" || dep.Status == "failed") {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("deployment %d did not reach terminal state in %s", deploymentID, timeout)
}
