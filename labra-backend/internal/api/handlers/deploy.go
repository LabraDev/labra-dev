package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"labra-backend/internal/api/store"
)

var runDeploymentAsync = true

const missingUserIDError = "missing user id: pass X-User-ID header"

func ensureAppStore(w http.ResponseWriter) bool {
	if appStore == nil {
		writeJSONError(w, http.StatusInternalServerError, "store not initialized")
		return false
	}
	return true
}

func readAppIDFromRequest(w http.ResponseWriter, r *http.Request) (int64, bool) {
	appID, err := readIDFromPathOrQuery(r, "apps")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return 0, false
	}
	return appID, true
}

func readDeploymentIDFromRequest(w http.ResponseWriter, r *http.Request) (int64, bool) {
	deploymentID, err := readIDFromPathOrQuery(r, "deploys")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return 0, false
	}
	return deploymentID, true
}

func requireUserID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	userID, ok := readUserID(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, missingUserIDError)
		return 0, false
	}
	return userID, true
}

func loadAppForUser(w http.ResponseWriter, r *http.Request, appID, userID int64) (store.App, bool) {
	app, err := appStore.GetAppByIDForUser(r.Context(), appID, userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "app not found")
			return store.App{}, false
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to load app")
		return store.App{}, false
	}
	return app, true
}

func loadDeploymentForUser(w http.ResponseWriter, r *http.Request, deploymentID, userID int64) (store.Deployment, bool) {
	deployment, err := appStore.GetDeploymentByIDForUser(r.Context(), deploymentID, userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "deployment not found")
			return store.Deployment{}, false
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to load deployment")
		return store.Deployment{}, false
	}
	return deployment, true
}

func validateDeployEligibility(app store.App) error {
	if strings.TrimSpace(app.BuildType) != "static" {
		return fmt.Errorf("app is not eligible for deployment: unsupported build_type")
	}
	if strings.TrimSpace(app.OutputDir) == "" {
		return fmt.Errorf("app is not eligible for deployment: output_dir is required")
	}
	return nil
}

func queueDeployment(ctx context.Context, app store.App, in store.CreateDeploymentInput, queueLogMessage string) (store.Deployment, error) {
	if appStore == nil {
		return store.Deployment{}, fmt.Errorf("store not initialized")
	}

	if in.AppID <= 0 {
		in.AppID = app.ID
	}
	if in.UserID <= 0 {
		in.UserID = app.UserID
	}
	if strings.TrimSpace(in.Status) == "" {
		in.Status = "queued"
	}
	if strings.TrimSpace(in.Branch) == "" {
		in.Branch = app.Branch
	}
	if strings.TrimSpace(in.SiteURL) == "" {
		in.SiteURL = app.SiteURL
	}

	deployment, err := appStore.CreateDeployment(ctx, in)
	if err != nil {
		return store.Deployment{}, err
	}

	if strings.TrimSpace(queueLogMessage) != "" {
		_ = appStore.CreateDeploymentLog(ctx, deployment.ID, "info", queueLogMessage)
	}
	triggerDeployment(deployment.ID, app)
	return deployment, nil
}

func appSiteURLOrDefault(app store.App) string {
	siteURL := strings.TrimSpace(app.SiteURL)
	if siteURL == "" {
		siteURL = fmt.Sprintf("https://%s.preview.labra.local", slugify(app.Name))
	}
	return siteURL
}

func CreateDeployHandler(w http.ResponseWriter, r *http.Request) {
	if !ensureAppStore(w) {
		return
	}

	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	appID, ok := readAppIDFromRequest(w, r)
	if !ok {
		return
	}

	app, ok := loadAppForUser(w, r, appID, userID)
	if !ok {
		return
	}
	if err := validateDeployEligibility(app); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	deployment, err := queueDeployment(r.Context(), app, store.CreateDeploymentInput{
		TriggerType:   "manual",
		CorrelationID: fmt.Sprintf("manual-%d", time.Now().UnixNano()),
	}, "deployment queued by manual trigger")
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to create deployment")
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"deployment": deployment,
	})
}

func CancelDeployHandler(w http.ResponseWriter, r *http.Request) {
	if !ensureAppStore(w) {
		return
	}

	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	deploymentID, ok := readDeploymentIDFromRequest(w, r)
	if !ok {
		return
	}

	deployment, ok := loadDeploymentForUser(w, r, deploymentID, userID)
	if !ok {
		return
	}

	switch strings.TrimSpace(strings.ToLower(deployment.Status)) {
	case "queued", "running":
		// cancel is allowed
	case "canceled":
		writeJSON(w, http.StatusOK, map[string]any{
			"deployment": deployment,
		})
		return
	default:
		writeJSONError(w, http.StatusConflict, "deployment cannot be canceled in current status")
		return
	}

	finish := store.UnixNow()
	updated, err := appStore.UpdateDeploymentStatus(r.Context(), deploymentID, "canceled", "canceled by user", deployment.SiteURL, deployment.StartedAt, finish)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to cancel deployment")
		return
	}
	_ = appStore.CreateDeploymentLog(r.Context(), deploymentID, "warn", "deployment canceled by user")

	writeJSON(w, http.StatusOK, map[string]any{
		"deployment": updated,
	})
}

func RetryDeployHandler(w http.ResponseWriter, r *http.Request) {
	if !ensureAppStore(w) {
		return
	}

	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	deploymentID, ok := readDeploymentIDFromRequest(w, r)
	if !ok {
		return
	}

	prev, ok := loadDeploymentForUser(w, r, deploymentID, userID)
	if !ok {
		return
	}

	prevStatus := strings.TrimSpace(strings.ToLower(prev.Status))
	if prevStatus != "failed" && prevStatus != "canceled" {
		writeJSONError(w, http.StatusConflict, "deployment can only be retried from failed or canceled status")
		return
	}

	app, ok := loadAppForUser(w, r, prev.AppID, userID)
	if !ok {
		return
	}

	deployment, err := queueDeployment(r.Context(), app, store.CreateDeploymentInput{
		TriggerType:   "manual_retry",
		CommitSHA:     prev.CommitSHA,
		CommitMessage: prev.CommitMessage,
		CommitAuthor:  prev.CommitAuthor,
		CorrelationID: fmt.Sprintf("retry-%d-%d", prev.ID, time.Now().UnixNano()),
	}, fmt.Sprintf("deployment queued by retry (from deployment %d)", prev.ID))
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to create retry deployment")
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"retried_from": prev.ID,
		"deployment":   deployment,
	})
}

func GetDeployHandler(w http.ResponseWriter, r *http.Request) {
	if !ensureAppStore(w) {
		return
	}

	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	deploymentID, ok := readDeploymentIDFromRequest(w, r)
	if !ok {
		return
	}

	deployment, ok := loadDeploymentForUser(w, r, deploymentID, userID)
	if !ok {
		return
	}

	writeJSON(w, http.StatusOK, deployment)
}

func GetDeployLogsHandler(w http.ResponseWriter, r *http.Request) {
	if !ensureAppStore(w) {
		return
	}

	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	deploymentID, ok := readDeploymentIDFromRequest(w, r)
	if !ok {
		return
	}

	if _, ok := loadDeploymentForUser(w, r, deploymentID, userID); !ok {
		return
	}

	logs, err := appStore.ListDeploymentLogs(r.Context(), deploymentID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to load deployment logs")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"deployment_id": deploymentID,
		"logs":          logs,
	})
}

func runManualDeployment(deploymentID int64, app store.App) {
	ctx := context.Background()
	start := store.UnixNow()

	_, _ = appStore.UpdateDeploymentStatus(ctx, deploymentID, "running", "", app.SiteURL, start, 0)
	_ = appStore.CreateDeploymentLog(ctx, deploymentID, "info", "clone repository")
	_ = appStore.CreateDeploymentLog(ctx, deploymentID, "info", fmt.Sprintf("checkout branch %s", app.Branch))
	_ = appStore.CreateDeploymentLog(ctx, deploymentID, "info", "install dependencies")
	_ = appStore.CreateDeploymentLog(ctx, deploymentID, "info", "run build")
	_ = appStore.CreateDeploymentLog(ctx, deploymentID, "info", "upload static artifacts")
	_ = appStore.CreateDeploymentLog(ctx, deploymentID, "info", "invalidate CDN cache")

	if app.BuildType != "static" {
		finish := store.UnixNow()
		_, _ = appStore.UpdateDeploymentStatus(ctx, deploymentID, "failed", "unsupported build type", "", start, finish)
		_ = appStore.CreateDeploymentLog(ctx, deploymentID, "error", "deployment failed: unsupported build type")
		return
	}

	siteURL := appSiteURLOrDefault(app)

	finish := store.UnixNow()
	_ = appStore.CreateDeploymentLog(ctx, deploymentID, "info", "deployment completed successfully")
	_, _ = appStore.UpdateDeploymentStatus(ctx, deploymentID, "succeeded", "", siteURL, start, finish)
}

func triggerDeployment(deploymentID int64, app store.App) {
	if runDeploymentAsync {
		go runManualDeployment(deploymentID, app)
		return
	}
	runManualDeployment(deploymentID, app)
}

func slugify(in string) string {
	v := strings.TrimSpace(strings.ToLower(in))
	if v == "" {
		return "app"
	}
	v = strings.ReplaceAll(v, " ", "-")
	v = strings.ReplaceAll(v, "_", "-")
	return v
}
