package handlers

import (
	"errors"
	"net/http"
	"time"

	"labra-backend/internal/api/store"
)

func GetDeployQueueStatusHandler(w http.ResponseWriter, r *http.Request) {
	if appStore == nil {
		writeJSONError(w, http.StatusInternalServerError, "store not initialized")
		return
	}

	userID, ok := readUserID(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing user id: pass X-User-ID header")
		return
	}

	deploymentID, err := readIDFromPathOrQuery(r, "deploys")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	if _, err := appStore.GetDeploymentByIDForUser(r.Context(), deploymentID, userID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "deployment not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to load deployment")
		return
	}

	job, err := appStore.GetDeploymentJobByDeploymentID(r.Context(), deploymentID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "deployment job not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to load deployment queue status")
		return
	}

	nextRetryIn := int64(0)
	now := store.UnixNow()
	if job.NextAttemptAt > now && (job.Status == "queued" || job.Status == "retrying") {
		nextRetryIn = job.NextAttemptAt - now
	}

	cfg := loadDeploymentQueueConfig()
	writeJSON(w, http.StatusOK, map[string]any{
		"deployment_id":           deploymentID,
		"job":                     job,
		"next_retry_in_seconds":   nextRetryIn,
		"worker":                  queueWorker,
		"queue_poll_interval_ms":  int(cfg.PollInterval / time.Millisecond),
		"worker_concurrency":      cfg.Concurrency,
		"configured_max_attempts": cfg.MaxAttempts,
	})
}
