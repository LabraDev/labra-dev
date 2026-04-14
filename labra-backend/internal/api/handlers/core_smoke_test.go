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
)

func TestCoreFlowSmoke(t *testing.T) {
	db := setupPhase4TestDB(t)
	defer db.Close()

	app := createTestApp(t, 1, "smoke-app", "owner/repo", "main")

	dep1ID := triggerManualDeployViaHandler(t, app.ID, app.UserID)
	waitForDeploymentTerminalStatusWithTimeout(t, dep1ID, app.UserID, 7*time.Second)
	assertQueueEndpointResponds(t, dep1ID, app.UserID)

	dep2ID := triggerManualDeployViaHandler(t, app.ID, app.UserID)
	waitForDeploymentTerminalStatusWithTimeout(t, dep2ID, app.UserID, 7*time.Second)
	assertQueueEndpointResponds(t, dep2ID, app.UserID)

	beforeRollback, err := appStore.GetAppByIDForUser(context.Background(), app.ID, app.UserID)
	if err != nil {
		t.Fatalf("get app before rollback: %v", err)
	}
	if beforeRollback.CurrentReleaseID == 0 {
		t.Fatalf("expected current release set before rollback")
	}

	rollbackReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/apps/%d/rollback", app.ID), bytes.NewBufferString(`{"reason":"core-smoke"}`))
	rollbackReq.Header.Set("Content-Type", "application/json")
	rollbackReq.Header.Set("X-User-ID", "1")
	rollbackRR := httptest.NewRecorder()
	CreateRollbackHandler(rollbackRR, rollbackReq)
	if rollbackRR.Code != http.StatusAccepted {
		t.Fatalf("rollback expected status %d, got %d body=%s", http.StatusAccepted, rollbackRR.Code, rollbackRR.Body.String())
	}

	var rollbackBody struct {
		Deployment struct {
			ID int64 `json:"id"`
		} `json:"deployment"`
		TargetReleaseID int64 `json:"target_release_id"`
	}
	if err := json.Unmarshal(rollbackRR.Body.Bytes(), &rollbackBody); err != nil {
		t.Fatalf("decode rollback response: %v", err)
	}
	if rollbackBody.Deployment.ID <= 0 || rollbackBody.TargetReleaseID <= 0 {
		t.Fatalf("unexpected rollback response payload: %s", rollbackRR.Body.String())
	}

	waitForDeploymentTerminalStatusWithTimeout(t, rollbackBody.Deployment.ID, app.UserID, 7*time.Second)

	afterRollback, err := appStore.GetAppByIDForUser(context.Background(), app.ID, app.UserID)
	if err != nil {
		t.Fatalf("get app after rollback: %v", err)
	}
	if afterRollback.CurrentReleaseID != rollbackBody.TargetReleaseID {
		t.Fatalf("expected current_release_id=%d after rollback, got %d", rollbackBody.TargetReleaseID, afterRollback.CurrentReleaseID)
	}
	if afterRollback.CurrentReleaseID == beforeRollback.CurrentReleaseID {
		t.Fatalf("expected rollback to change release pointer")
	}

	healthReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/apps/%d/health", app.ID), nil)
	healthReq.Header.Set("X-User-ID", "1")
	healthRR := httptest.NewRecorder()
	GetAppHealthSummaryHandler(healthRR, healthReq)
	if healthRR.Code != http.StatusOK {
		t.Fatalf("health expected status %d, got %d body=%s", http.StatusOK, healthRR.Code, healthRR.Body.String())
	}

	var healthBody struct {
		LatestDeployStatus string `json:"latest_deploy_status"`
	}
	if err := json.Unmarshal(healthRR.Body.Bytes(), &healthBody); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	if healthBody.LatestDeployStatus == "" {
		t.Fatalf("expected latest_deploy_status in health response")
	}

	obsReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/apps/%d/observability", app.ID), nil)
	obsReq.Header.Set("X-User-ID", "1")
	obsRR := httptest.NewRecorder()
	GetAppObservabilitySummaryHandler(obsRR, obsReq)
	if obsRR.Code != http.StatusOK {
		t.Fatalf("observability expected status %d, got %d body=%s", http.StatusOK, obsRR.Code, obsRR.Body.String())
	}

	var obsBody struct {
		ReleaseCount int64 `json:"release_count"`
	}
	if err := json.Unmarshal(obsRR.Body.Bytes(), &obsBody); err != nil {
		t.Fatalf("decode observability response: %v", err)
	}
	if obsBody.ReleaseCount < 2 {
		t.Fatalf("expected release_count >= 2, got %d", obsBody.ReleaseCount)
	}
}

func assertQueueEndpointResponds(t *testing.T, deploymentID, userID int64) {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/deploys/%d/queue", deploymentID), nil)
	req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
	rr := httptest.NewRecorder()
	GetDeployQueueStatusHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("queue expected status %d, got %d body=%s", http.StatusOK, rr.Code, rr.Body.String())
	}
}
