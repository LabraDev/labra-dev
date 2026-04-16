package handlers

import (
	"net/http"
)

func GetAppDeploysHandler(w http.ResponseWriter, r *http.Request) {
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

	deployments, err := appStore.ListDeploymentsByAppForUser(r.Context(), app.ID, userID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to load app deployments")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"app_id":      app.ID,
		"app_name":    app.Name,
		"repo":        app.RepoFullName,
		"branch":      app.Branch,
		"deployments": deployments,
	})
}
