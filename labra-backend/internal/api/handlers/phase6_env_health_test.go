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

	"labra-backend/internal/api/store"
)

func TestAppEnvVarCRUDAndMasking(t *testing.T) {
	db := setupPhase4TestDB(t)
	defer db.Close()

	app := createTestApp(t, 1, "demo", "owner/repo", "main")

	secretTrue := true
	createSecret := createAppEnvVarRequest{
		Key:      "API_TOKEN",
		Value:    "super-secret-token",
		IsSecret: &secretTrue,
	}
	createdSecret := createEnvVarViaHandler(t, app.ID, 1, createSecret)
	if !createdSecret.Masked || createdSecret.Value != "********" {
		t.Fatalf("expected secret env var to be masked, got %#v", createdSecret)
	}

	createPublic := createAppEnvVarRequest{
		Key:   "NODE_ENV",
		Value: "production",
	}
	createdPublic := createEnvVarViaHandler(t, app.ID, 1, createPublic)
	if createdPublic.Masked || createdPublic.Value != "production" {
		t.Fatalf("expected non-secret env var to be visible, got %#v", createdPublic)
	}

	listReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/apps/%d/env-vars", app.ID), nil)
	listReq.Header.Set("X-User-ID", "1")
	listRR := httptest.NewRecorder()
	ListAppEnvVarsHandler(listRR, listReq)
	if listRR.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, listRR.Code, listRR.Body.String())
	}

	var listBody struct {
		EnvVars []appEnvVarResponse `json:"env_vars"`
	}
	if err := json.Unmarshal(listRR.Body.Bytes(), &listBody); err != nil {
		t.Fatalf("failed to decode list response: %v", err)
	}
	if len(listBody.EnvVars) != 2 {
		t.Fatalf("expected 2 env vars, got %d", len(listBody.EnvVars))
	}

	secretVar := findEnvVarByKey(t, listBody.EnvVars, "API_TOKEN")
	if !secretVar.Masked || secretVar.Value != "********" {
		t.Fatalf("expected secret value masked in list response, got %#v", secretVar)
	}
	publicVar := findEnvVarByKey(t, listBody.EnvVars, "NODE_ENV")
	if publicVar.Masked || publicVar.Value != "production" {
		t.Fatalf("expected public value visible in list response, got %#v", publicVar)
	}

	updatedValue := "rotated-secret"
	patchBody, _ := json.Marshal(updateAppEnvVarRequest{Value: &updatedValue})
	patchReq := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/v1/apps/%d/env-vars/%d", app.ID, createdSecret.ID), bytes.NewReader(patchBody))
	patchReq.Header.Set("Content-Type", "application/json")
	patchReq.Header.Set("X-User-ID", "1")
	patchRR := httptest.NewRecorder()
	PatchAppEnvVarHandler(patchRR, patchReq)
	if patchRR.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, patchRR.Code, patchRR.Body.String())
	}

	var patched appEnvVarResponse
	if err := json.Unmarshal(patchRR.Body.Bytes(), &patched); err != nil {
		t.Fatalf("failed to decode patch response: %v", err)
	}
	if !patched.Masked || patched.Value != "********" {
		t.Fatalf("expected patched secret value to stay masked, got %#v", patched)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/v1/apps/%d/env-vars/%d", app.ID, createdPublic.ID), nil)
	deleteReq.Header.Set("X-User-ID", "1")
	deleteRR := httptest.NewRecorder()
	DeleteAppEnvVarHandler(deleteRR, deleteReq)
	if deleteRR.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusNoContent, deleteRR.Code, deleteRR.Body.String())
	}

	listReq2 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/apps/%d/env-vars", app.ID), nil)
	listReq2.Header.Set("X-User-ID", "1")
	listRR2 := httptest.NewRecorder()
	ListAppEnvVarsHandler(listRR2, listReq2)
	if listRR2.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, listRR2.Code, listRR2.Body.String())
	}
	if err := json.Unmarshal(listRR2.Body.Bytes(), &listBody); err != nil {
		t.Fatalf("failed to decode second list response: %v", err)
	}
	if len(listBody.EnvVars) != 1 || listBody.EnvVars[0].Key != "API_TOKEN" {
		t.Fatalf("expected only API_TOKEN to remain, got %#v", listBody.EnvVars)
	}
}

func TestCreateAppEnvVarRejectsDuplicateKey(t *testing.T) {
	db := setupPhase4TestDB(t)
	defer db.Close()

	app := createTestApp(t, 1, "demo", "owner/repo", "main")
	_ = createEnvVarViaHandler(t, app.ID, 1, createAppEnvVarRequest{Key: "API_TOKEN", Value: "first"})

	body, _ := json.Marshal(createAppEnvVarRequest{Key: "API_TOKEN", Value: "second"})
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/apps/%d/env-vars", app.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "1")
	rr := httptest.NewRecorder()
	CreateAppEnvVarHandler(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusConflict, rr.Code, rr.Body.String())
	}
}

func TestRunManualDeploymentInjectsEnvVarsAndRecordsMetrics(t *testing.T) {
	db := setupPhase4TestDB(t)
	defer db.Close()

	app := createTestApp(t, 1, "demo", "owner/repo", "main")
	_, err := appStore.CreateAppEnvVar(context.Background(), app.ID, app.UserID, store.CreateAppEnvVarInput{Key: "NODE_ENV", Value: "production", IsSecret: false})
	if err != nil {
		t.Fatalf("create non-secret env var: %v", err)
	}
	_, err = appStore.CreateAppEnvVar(context.Background(), app.ID, app.UserID, store.CreateAppEnvVarInput{Key: "API_TOKEN", Value: "super-secret", IsSecret: true})
	if err != nil {
		t.Fatalf("create secret env var: %v", err)
	}

	dep, err := appStore.CreateDeployment(context.Background(), store.CreateDeploymentInput{
		AppID:         app.ID,
		UserID:        app.UserID,
		Status:        "queued",
		TriggerType:   "manual",
		Branch:        app.Branch,
		CorrelationID: "manual-test",
	})
	if err != nil {
		t.Fatalf("create deployment: %v", err)
	}

	runManualDeployment(dep.ID, app)

	logs, err := appStore.ListDeploymentLogs(context.Background(), dep.ID)
	if err != nil {
		t.Fatalf("list deployment logs: %v", err)
	}

	foundInjectionLog := false
	for _, logLine := range logs {
		if strings.Contains(logLine.Message, "inject 2 env vars (1 secret)") {
			foundInjectionLog = true
			break
		}
	}
	if !foundInjectionLog {
		t.Fatalf("expected env var injection log, got logs=%#v", logs)
	}

	updatedDep, err := appStore.GetDeploymentByIDForUser(context.Background(), dep.ID, app.UserID)
	if err != nil {
		t.Fatalf("get updated deployment: %v", err)
	}
	if updatedDep.Status != "succeeded" {
		t.Fatalf("expected deployment status succeeded, got %q", updatedDep.Status)
	}

	metrics, err := appStore.GetAppHealthMetricsForUser(context.Background(), app.ID, app.UserID)
	if err != nil {
		t.Fatalf("get app health metrics: %v", err)
	}
	if metrics.SuccessCount != 1 || metrics.FailureCount != 0 {
		t.Fatalf("expected metrics success=1 failure=0, got %#v", metrics)
	}
}

func TestGetAppHealthSummaryHandlerReturnsExpectedShape(t *testing.T) {
	db := setupPhase4TestDB(t)
	defer db.Close()

	app := createTestApp(t, 1, "demo", "owner/repo", "main")
	_, err := appStore.CreateAppEnvVar(context.Background(), app.ID, app.UserID, store.CreateAppEnvVarInput{Key: "LABRA_ALARM_STATE", Value: "ok", IsSecret: false})
	if err != nil {
		t.Fatalf("create alarm env var: %v", err)
	}

	dep1, err := appStore.CreateDeployment(context.Background(), store.CreateDeploymentInput{
		AppID:         app.ID,
		UserID:        app.UserID,
		Status:        "queued",
		TriggerType:   "manual",
		CorrelationID: "dep-success",
	})
	if err != nil {
		t.Fatalf("create deployment #1: %v", err)
	}
	if _, err := appStore.UpdateDeploymentStatus(context.Background(), dep1.ID, "succeeded", "", "https://demo.preview.labra.local", 100, 110); err != nil {
		t.Fatalf("mark deployment #1 succeeded: %v", err)
	}
	if err := appStore.RecordAppDeploymentOutcome(context.Background(), app.ID, "succeeded", 100, 110, "manual"); err != nil {
		t.Fatalf("record deployment #1 outcome: %v", err)
	}

	dep2, err := appStore.CreateDeployment(context.Background(), store.CreateDeploymentInput{
		AppID:         app.ID,
		UserID:        app.UserID,
		Status:        "queued",
		TriggerType:   "manual",
		CorrelationID: "dep-failed",
	})
	if err != nil {
		t.Fatalf("create deployment #2: %v", err)
	}
	if _, err := appStore.UpdateDeploymentStatus(context.Background(), dep2.ID, "failed", "build failed", "", 120, 130); err != nil {
		t.Fatalf("mark deployment #2 failed: %v", err)
	}
	if err := appStore.RecordAppDeploymentOutcome(context.Background(), app.ID, "failed", 120, 130, "manual"); err != nil {
		t.Fatalf("record deployment #2 outcome: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/apps/%d/health", app.ID), nil)
	req.Header.Set("X-User-ID", "1")
	rr := httptest.NewRecorder()
	GetAppHealthSummaryHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var body appHealthResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body.LatestDeploy == nil || body.LatestDeploy.ID != dep2.ID {
		t.Fatalf("expected latest deployment id %d, got %#v", dep2.ID, body.LatestDeploy)
	}
	if body.LatestDeployStatus != "failed" {
		t.Fatalf("expected latest deploy status failed, got %q", body.LatestDeployStatus)
	}
	if body.LastSuccessfulDeploy == nil || body.LastSuccessfulDeploy.ID != dep1.ID {
		t.Fatalf("expected last successful deployment id %d, got %#v", dep1.ID, body.LastSuccessfulDeploy)
	}
	if body.Metrics.SuccessCount != 1 || body.Metrics.FailureCount != 1 || body.Metrics.TotalCount != 2 {
		t.Fatalf("unexpected metrics payload: %#v", body.Metrics)
	}
	if body.Metrics.SuccessRate != 50 {
		t.Fatalf("expected success rate 50, got %v", body.Metrics.SuccessRate)
	}
	if body.CurrentURL != "https://demo.preview.labra.local" {
		t.Fatalf("expected current URL to fallback to last successful URL, got %q", body.CurrentURL)
	}
	if body.AlarmState == nil || *body.AlarmState != "ok" {
		t.Fatalf("expected alarm_state=ok, got %#v", body.AlarmState)
	}
	if body.HealthIndicator != "unhealthy" {
		t.Fatalf("expected health indicator unhealthy, got %q", body.HealthIndicator)
	}
}

func createEnvVarViaHandler(t *testing.T, appID, userID int64, body createAppEnvVarRequest) appEnvVarResponse {
	t.Helper()

	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/apps/%d/env-vars", appID), bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
	rr := httptest.NewRecorder()
	CreateAppEnvVarHandler(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusCreated, rr.Code, rr.Body.String())
	}

	var response appEnvVarResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode create env var response: %v", err)
	}
	return response
}

func findEnvVarByKey(t *testing.T, vars []appEnvVarResponse, key string) appEnvVarResponse {
	t.Helper()
	for _, envVar := range vars {
		if envVar.Key == key {
			return envVar
		}
	}
	t.Fatalf("env var %q not found in %#v", key, vars)
	return appEnvVarResponse{}
}
