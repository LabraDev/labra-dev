package handlers

import (
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"

	"labra-backend/internal/api/store"
)

type deploymentDurationPoint struct {
	DeploymentID    int64  `json:"deployment_id"`
	Status          string `json:"status"`
	TriggerType     string `json:"trigger_type"`
	DurationSeconds int64  `json:"duration_seconds"`
	FinishedAt      int64  `json:"finished_at"`
}

type observabilitySummaryResponse struct {
	AppID             int64                     `json:"app_id"`
	CurrentReleaseID  int64                     `json:"current_release_id,omitempty"`
	ReleaseCount      int                       `json:"release_count"`
	StatusCounts      map[string]int            `json:"status_counts"`
	TriggerCounts     map[string]int            `json:"trigger_counts"`
	RecentDurations   []deploymentDurationPoint `json:"recent_durations"`
	RecentFailures    []store.Deployment        `json:"recent_failures"`
	RecentRollbacks   []store.RollbackEvent     `json:"recent_rollbacks"`
	Health            healthMetricsResponse     `json:"health"`
	HealthIndicator   string                    `json:"health_indicator"`
	AlarmState        *string                   `json:"alarm_state,omitempty"`
	CloudWatchEnabled bool                      `json:"cloudwatch_enabled"`
}

func GetAppObservabilitySummaryHandler(w http.ResponseWriter, r *http.Request) {
	if appStore == nil {
		writeJSONError(w, http.StatusInternalServerError, "store not initialized")
		return
	}

	userID, ok := readUserID(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing user id: pass X-User-ID header")
		return
	}

	appID, err := readIDFromPathOrQuery(r, "apps")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	app, err := appStore.GetAppByIDForUser(r.Context(), appID, userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "app not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to load app")
		return
	}

	recentDeploys, err := appStore.ListRecentDeploymentsByAppForUser(r.Context(), appID, userID, 30)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to load recent deployments")
		return
	}

	releases, err := appStore.ListReleaseVersionsByAppForUser(r.Context(), appID, userID, 200)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to load releases")
		return
	}

	rollbacks, err := appStore.ListRollbackEventsByAppForUser(r.Context(), appID, userID, 20)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to load rollback history")
		return
	}

	metrics, err := appStore.GetAppHealthMetricsForUser(r.Context(), appID, userID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to load health metrics")
		return
	}

	envVars, envErr := appStore.ListAppEnvVarsForApp(r.Context(), appID)
	if envErr != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to load app env vars")
		return
	}
	alarmState := resolveAlarmState(envVars)

	statusCounts := map[string]int{}
	triggerCounts := map[string]int{}
	durations := make([]deploymentDurationPoint, 0, len(recentDeploys))
	failures := make([]store.Deployment, 0)

	for _, dep := range recentDeploys {
		statusKey := strings.TrimSpace(strings.ToLower(dep.Status))
		if statusKey == "" {
			statusKey = "unknown"
		}
		statusCounts[statusKey]++

		triggerKey := strings.TrimSpace(strings.ToLower(dep.TriggerType))
		if triggerKey == "" {
			triggerKey = "unknown"
		}
		triggerCounts[triggerKey]++

		duration := int64(0)
		if dep.StartedAt > 0 && dep.FinishedAt >= dep.StartedAt {
			duration = dep.FinishedAt - dep.StartedAt
		}
		durations = append(durations, deploymentDurationPoint{
			DeploymentID:    dep.ID,
			Status:          dep.Status,
			TriggerType:     dep.TriggerType,
			DurationSeconds: duration,
			FinishedAt:      dep.FinishedAt,
		})

		if strings.EqualFold(dep.Status, "failed") && strings.TrimSpace(dep.FailureReason) != "" && len(failures) < 8 {
			failures = append(failures, dep)
		}
	}

	total := metrics.SuccessCount + metrics.FailureCount
	successRate := 0.0
	avgDuration := 0.0
	if total > 0 {
		successRate = (float64(metrics.SuccessCount) / float64(total)) * 100
		avgDuration = float64(metrics.TotalDuration) / float64(total)
	}

	latestSummary := healthDeploymentSummary{}
	if len(recentDeploys) > 0 {
		latestSummary = healthDeploymentSummary{Status: recentDeploys[0].Status}
	}

	writeJSON(w, http.StatusOK, observabilitySummaryResponse{
		AppID:            appID,
		CurrentReleaseID: app.CurrentReleaseID,
		ReleaseCount:     len(releases),
		StatusCounts:     statusCounts,
		TriggerCounts:    triggerCounts,
		RecentDurations:  durations,
		RecentFailures:   failures,
		RecentRollbacks:  rollbacks,
		Health: healthMetricsResponse{
			SuccessCount:    metrics.SuccessCount,
			FailureCount:    metrics.FailureCount,
			TotalCount:      total,
			SuccessRate:     successRate,
			AverageDuration: avgDuration,
			LatestDuration:  metrics.LatestDuration,
			RollbackCount:   metrics.RollbackCount,
			LastDeployAt:    metrics.LastDeployAt,
			LastSuccessAt:   metrics.LastSuccessAt,
			LastFailureAt:   metrics.LastFailureAt,
			TotalDuration:   metrics.TotalDuration,
		},
		HealthIndicator:   computeHealthIndicator(&latestSummary),
		AlarmState:        alarmState,
		CloudWatchEnabled: isCloudWatchQueryEnabled(),
	})
}

func QueryAppObservabilityLogsHandler(w http.ResponseWriter, r *http.Request) {
	if appStore == nil {
		writeJSONError(w, http.StatusInternalServerError, "store not initialized")
		return
	}

	userID, ok := readUserID(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing user id: pass X-User-ID header")
		return
	}

	appID, err := readIDFromPathOrQuery(r, "apps")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	if _, err := appStore.GetAppByIDForUser(r.Context(), appID, userID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "app not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to load app")
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		writeJSONError(w, http.StatusBadRequest, "q query parameter is required")
		return
	}

	source := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("source")))
	if source == "" {
		source = "local"
	}

	limit := readIntQueryParam(r, "limit", 20, 1, 200)
	provider := "local-sqlite"
	if source == "cloudwatch" {
		if !isCloudWatchQueryEnabled() {
			writeJSON(w, http.StatusOK, map[string]any{
				"app_id":    appID,
				"query":     query,
				"provider":  "cloudwatch",
				"available": false,
				"reason":    "set CLOUDWATCH_QUERY_ENABLED=true to enable cloudwatch-compatible mode",
				"hits":      []store.LogQueryHit{},
			})
			return
		}
		provider = "cloudwatch-compat-local"
	}

	hits, err := appStore.QueryDeploymentLogsByAppForUser(r.Context(), appID, userID, query, limit)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to execute log query")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"app_id":    appID,
		"query":     query,
		"provider":  provider,
		"available": true,
		"hits":      hits,
	})
}

func isCloudWatchQueryEnabled() bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv("CLOUDWATCH_QUERY_ENABLED")))
	return raw == "1" || raw == "true" || raw == "yes" || raw == "on"
}

func readIntQueryParam(r *http.Request, key string, defaultValue, minValue, maxValue int) int {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return defaultValue
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return defaultValue
	}
	if parsed < minValue {
		return minValue
	}
	if parsed > maxValue {
		return maxValue
	}
	return parsed
}
