package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"labra-backend/internal/api/store"
)

func TestRunManualDeploymentCreatesReleaseAndAppliesRetention(t *testing.T) {
	t.Setenv("RELEASE_RETENTION_LIMIT", "2")

	db := setupPhase4TestDB(t)
	defer db.Close()

	app := createTestApp(t, 1, "demo", "owner/repo", "main")
	deploymentIDs := make([]int64, 0, 3)

	for i := 0; i < 3; i++ {
		dep, err := appStore.CreateDeployment(context.Background(), store.CreateDeploymentInput{
			AppID:         app.ID,
			UserID:        app.UserID,
			Status:        "queued",
			TriggerType:   "manual",
			Branch:        app.Branch,
			CorrelationID: fmt.Sprintf("manual-release-%d", i+1),
		})
		if err != nil {
			t.Fatalf("create deployment %d: %v", i+1, err)
		}
		deploymentIDs = append(deploymentIDs, dep.ID)
		runManualDeployment(dep.ID, app)
	}

	releases, err := appStore.ListReleaseVersionsByAppForUser(context.Background(), app.ID, app.UserID, 10)
	if err != nil {
		t.Fatalf("list releases: %v", err)
	}
	if len(releases) != 3 {
		t.Fatalf("expected 3 releases, got %d", len(releases))
	}

	currentApp, err := appStore.GetAppByIDForUser(context.Background(), app.ID, app.UserID)
	if err != nil {
		t.Fatalf("get app: %v", err)
	}
	if currentApp.CurrentReleaseID == 0 || currentApp.CurrentReleaseID != releases[0].ID {
		t.Fatalf("expected current release id %d, got %d", releases[0].ID, currentApp.CurrentReleaseID)
	}

	if !releases[0].IsRetained || !releases[1].IsRetained {
		t.Fatalf("expected latest two releases retained, got %#v", releases)
	}
	if releases[2].IsRetained {
		t.Fatalf("expected oldest release pruned from retention set")
	}

	for _, depID := range deploymentIDs {
		dep, err := appStore.GetDeploymentByIDForUser(context.Background(), depID, app.UserID)
		if err != nil {
			t.Fatalf("get deployment %d: %v", depID, err)
		}
		if dep.ReleaseID == 0 {
			t.Fatalf("expected deployment %d to be linked to a release", depID)
		}
	}
}

func TestCreateRollbackHandlerSwitchesCurrentReleaseAndRecordsEvent(t *testing.T) {
	db := setupPhase4TestDB(t)
	defer db.Close()

	app := createTestApp(t, 1, "demo", "owner/repo", "main")

	dep1, err := appStore.CreateDeployment(context.Background(), store.CreateDeploymentInput{
		AppID:         app.ID,
		UserID:        app.UserID,
		Status:        "queued",
		TriggerType:   "manual",
		Branch:        app.Branch,
		CorrelationID: "rollback-base-1",
	})
	if err != nil {
		t.Fatalf("create first deployment: %v", err)
	}
	runManualDeployment(dep1.ID, app)

	dep2, err := appStore.CreateDeployment(context.Background(), store.CreateDeploymentInput{
		AppID:         app.ID,
		UserID:        app.UserID,
		Status:        "queued",
		TriggerType:   "manual",
		Branch:        app.Branch,
		CorrelationID: "rollback-base-2",
	})
	if err != nil {
		t.Fatalf("create second deployment: %v", err)
	}
	runManualDeployment(dep2.ID, app)

	releases, err := appStore.ListReleaseVersionsByAppForUser(context.Background(), app.ID, app.UserID, 10)
	if err != nil {
		t.Fatalf("list releases: %v", err)
	}
	if len(releases) < 2 {
		t.Fatalf("expected at least 2 releases, got %d", len(releases))
	}

	body := bytes.NewBufferString(`{"reason":"hotfix regression"}`)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/apps/%d/rollback", app.ID), body)
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
		FromReleaseID   int64 `json:"from_release_id"`
		TargetReleaseID int64 `json:"target_release_id"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode rollback response: %v", err)
	}

	if response.FromReleaseID == 0 || response.TargetReleaseID == 0 {
		t.Fatalf("expected non-zero release ids in rollback response: %#v", response)
	}

	waitForDeploymentTerminalStatus(t, response.Deployment.ID, app.UserID)

	rollbackDeployment, err := appStore.GetDeploymentByIDForUser(context.Background(), response.Deployment.ID, app.UserID)
	if err != nil {
		t.Fatalf("get rollback deployment: %v", err)
	}
	if rollbackDeployment.Status != "succeeded" {
		t.Fatalf("expected rollback deployment succeeded, got %q", rollbackDeployment.Status)
	}
	if rollbackDeployment.TriggerType != "rollback" {
		t.Fatalf("expected trigger_type=rollback, got %q", rollbackDeployment.TriggerType)
	}
	if rollbackDeployment.ReleaseID != response.TargetReleaseID {
		t.Fatalf("expected rollback deployment release_id=%d, got %d", response.TargetReleaseID, rollbackDeployment.ReleaseID)
	}

	currentApp, err := appStore.GetAppByIDForUser(context.Background(), app.ID, app.UserID)
	if err != nil {
		t.Fatalf("get app after rollback: %v", err)
	}
	if currentApp.CurrentReleaseID != response.TargetReleaseID {
		t.Fatalf("expected app current release to be %d, got %d", response.TargetReleaseID, currentApp.CurrentReleaseID)
	}

	rollbacks, err := appStore.ListRollbackEventsByAppForUser(context.Background(), app.ID, app.UserID, 10)
	if err != nil {
		t.Fatalf("list rollback events: %v", err)
	}
	if len(rollbacks) == 0 {
		t.Fatalf("expected rollback event to be recorded")
	}
	if rollbacks[0].ToReleaseID != response.TargetReleaseID {
		t.Fatalf("expected rollback event target=%d, got %d", response.TargetReleaseID, rollbacks[0].ToReleaseID)
	}

	metrics, err := appStore.GetAppHealthMetricsForUser(context.Background(), app.ID, app.UserID)
	if err != nil {
		t.Fatalf("get health metrics: %v", err)
	}
	if metrics.RollbackCount < 1 {
		t.Fatalf("expected rollback_count >= 1, got %d", metrics.RollbackCount)
	}
}

func TestObservabilitySummaryAndLogQueryHandlers(t *testing.T) {
	t.Setenv("CLOUDWATCH_QUERY_ENABLED", "false")

	db := setupPhase4TestDB(t)
	defer db.Close()

	app := createTestApp(t, 1, "obs", "owner/repo", "main")

	successDep, err := appStore.CreateDeployment(context.Background(), store.CreateDeploymentInput{
		AppID:         app.ID,
		UserID:        app.UserID,
		Status:        "queued",
		TriggerType:   "manual",
		Branch:        app.Branch,
		CorrelationID: "obs-success",
	})
	if err != nil {
		t.Fatalf("create success deployment: %v", err)
	}
	runManualDeployment(successDep.ID, app)

	failedDep, err := appStore.CreateDeployment(context.Background(), store.CreateDeploymentInput{
		AppID:         app.ID,
		UserID:        app.UserID,
		Status:        "queued",
		TriggerType:   "manual",
		Branch:        app.Branch,
		CorrelationID: "obs-failed",
	})
	if err != nil {
		t.Fatalf("create failed deployment: %v", err)
	}
	failingApp := app
	failingApp.BuildType = "container"
	runManualDeployment(failedDep.ID, failingApp)

	summaryReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/apps/%d/observability", app.ID), nil)
	summaryReq.Header.Set("X-User-ID", "1")
	summaryRR := httptest.NewRecorder()
	GetAppObservabilitySummaryHandler(summaryRR, summaryReq)

	if summaryRR.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, summaryRR.Code, summaryRR.Body.String())
	}

	var summary observabilitySummaryResponse
	if err := json.Unmarshal(summaryRR.Body.Bytes(), &summary); err != nil {
		t.Fatalf("decode summary response: %v", err)
	}
	if summary.ReleaseCount < 1 {
		t.Fatalf("expected release_count >= 1, got %d", summary.ReleaseCount)
	}
	if summary.StatusCounts["succeeded"] < 1 || summary.StatusCounts["failed"] < 1 {
		t.Fatalf("expected status counts for succeeded/failed, got %#v", summary.StatusCounts)
	}
	if summary.Health.TotalCount < 2 {
		t.Fatalf("expected total health count >= 2, got %d", summary.Health.TotalCount)
	}

	queryReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/apps/%d/observability/log-query?q=build&limit=20", app.ID), nil)
	queryReq.Header.Set("X-User-ID", "1")
	queryRR := httptest.NewRecorder()
	QueryAppObservabilityLogsHandler(queryRR, queryReq)
	if queryRR.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, queryRR.Code, queryRR.Body.String())
	}

	var queryBody struct {
		Available bool                `json:"available"`
		Hits      []store.LogQueryHit `json:"hits"`
	}
	if err := json.Unmarshal(queryRR.Body.Bytes(), &queryBody); err != nil {
		t.Fatalf("decode log query response: %v", err)
	}
	if !queryBody.Available {
		t.Fatalf("expected local log query to be available")
	}
	if len(queryBody.Hits) == 0 {
		t.Fatalf("expected non-empty log query hits")
	}

	cloudReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/apps/%d/observability/log-query?q=build&source=cloudwatch", app.ID), nil)
	cloudReq.Header.Set("X-User-ID", "1")
	cloudRR := httptest.NewRecorder()
	QueryAppObservabilityLogsHandler(cloudRR, cloudReq)
	if cloudRR.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, cloudRR.Code, cloudRR.Body.String())
	}

	var cloudBody struct {
		Available bool   `json:"available"`
		Provider  string `json:"provider"`
	}
	if err := json.Unmarshal(cloudRR.Body.Bytes(), &cloudBody); err != nil {
		t.Fatalf("decode cloudwatch query response: %v", err)
	}
	if cloudBody.Available {
		t.Fatalf("expected cloudwatch query disabled in test environment")
	}
	if cloudBody.Provider != "cloudwatch" {
		t.Fatalf("expected provider cloudwatch, got %q", cloudBody.Provider)
	}
}

func waitForDeploymentTerminalStatus(t *testing.T, deploymentID, userID int64) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		dep, err := appStore.GetDeploymentByIDForUser(context.Background(), deploymentID, userID)
		if err == nil && (dep.Status == "succeeded" || dep.Status == "failed") {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("deployment %d did not reach terminal state in time", deploymentID)
}
