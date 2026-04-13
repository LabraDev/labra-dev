package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

var githubWebhookSecret string

type githubPushEvent struct {
	Ref        string `json:"ref"`
	After      string `json:"after"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	HeadCommit struct {
		ID      string `json:"id"`
		Message string `json:"message"`
		Author  struct {
			Name string `json:"name"`
		} `json:"author"`
	} `json:"head_commit"`
}

type webhookResolution struct {
	RepoFullName string
	Branch       string
	MatchedApps  int
	EligibleApps []map[string]any
	Ignored      bool
	Reason       string
}

var resolvePushEvent = resolvePushEventDefault
var dedupeEligibleApps = dedupeEligibleAppsDefault
var enqueueEligibleApps = enqueueEligibleAppsDefault

func InitWebhook(secret string) {
	githubWebhookSecret = strings.TrimSpace(secret)
}

func GitHubWebhookHandler(w http.ResponseWriter, r *http.Request) {
	if appStore == nil {
		writeJSONError(w, http.StatusInternalServerError, "store not initialized")
		return
	}
	if githubWebhookSecret == "" {
		writeJSONError(w, http.StatusInternalServerError, "github webhook secret is not configured")
		return
	}

	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "unable to read request body")
		return
	}

	signature := strings.TrimSpace(r.Header.Get("X-Hub-Signature-256"))
	if !isValidGitHubSignature(githubWebhookSecret, rawBody, signature) {
		writeJSONError(w, http.StatusUnauthorized, "invalid webhook signature")
		return
	}

	eventType := strings.ToLower(strings.TrimSpace(r.Header.Get("X-GitHub-Event")))
	deliveryID := strings.TrimSpace(r.Header.Get("X-GitHub-Delivery"))

	if eventType != "push" {
		writeJSON(w, http.StatusAccepted, map[string]any{
			"accepted":    true,
			"ignored":     true,
			"delivery_id": deliveryID,
			"event_type":  eventType,
			"reason":      "event type is not supported",
		})
		return
	}

	var payload githubPushEvent
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid push payload")
		return
	}

	resolution, err := resolvePushEvent(r, payload)
	if err != nil {
		writeWebhookError(w, err)
		return
	}

	dedupedApps, duplicateCount, err := dedupeEligibleApps(r, deliveryID, eventType, payload, resolution.EligibleApps)
	if err != nil {
		writeWebhookError(w, err)
		return
	}

	triggeredDeployments, err := enqueueEligibleApps(r, payload, dedupedApps)
	if err != nil {
		writeWebhookError(w, err)
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"accepted":       true,
		"ignored":        resolution.Ignored || len(dedupedApps) == 0,
		"delivery_id":    deliveryID,
		"event_type":     eventType,
		"repo_full_name": resolution.RepoFullName,
		"branch":         resolution.Branch,
		"commit_sha":     strings.TrimSpace(payload.After),
		"commit_message": strings.TrimSpace(payload.HeadCommit.Message),
		"commit_author":  strings.TrimSpace(payload.HeadCommit.Author.Name),
		"matched_apps":   resolution.MatchedApps,
		"eligible_apps":  dedupedApps,
		"duplicate_count": duplicateCount,
		"triggered_count": len(triggeredDeployments),
		"triggered_deployments": triggeredDeployments,
		"reason":         resolution.Reason,
	})
}

func isValidGitHubSignature(secret string, body []byte, signature string) bool {
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}

	sigHex := strings.TrimPrefix(signature, "sha256=")
	expected, err := hex.DecodeString(sigHex)
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	computed := mac.Sum(nil)
	return hmac.Equal(computed, expected)
}

func extractBranch(ref string) (string, bool) {
	const prefix = "refs/heads/"
	if !strings.HasPrefix(ref, prefix) {
		return "", false
	}

	branch := strings.TrimSpace(strings.TrimPrefix(ref, prefix))
	if branch == "" {
		return "", false
	}

	return branch, true
}

func resolvePushEventDefault(_ *http.Request, payload githubPushEvent) (webhookResolution, error) {
	repoFullName := strings.ToLower(strings.TrimSpace(payload.Repository.FullName))
	if repoFullName == "" {
		return webhookResolution{}, errWebhook("repository.full_name is required")
	}

	branch, ok := extractBranch(payload.Ref)
	if !ok {
		return webhookResolution{
			RepoFullName: repoFullName,
			Ignored:      true,
			Reason:       "ref is not a branch push",
		}, nil
	}

	return webhookResolution{
		RepoFullName: repoFullName,
		Branch:       branch,
		Ignored:      true,
		Reason:       "routing not enabled",
	}, nil
}

type webhookErr string

func (e webhookErr) Error() string { return string(e) }

func errWebhook(msg string) error {
	return webhookErr(msg)
}

func writeWebhookError(w http.ResponseWriter, err error) {
	if _, ok := err.(webhookErr); ok {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSONError(w, http.StatusInternalServerError, err.Error())
}

func dedupeEligibleAppsDefault(_ *http.Request, _ string, _ string, _ githubPushEvent, eligibleApps []map[string]any) ([]map[string]any, int, error) {
	return eligibleApps, 0, nil
}

func enqueueEligibleAppsDefault(_ *http.Request, _ githubPushEvent, _ []map[string]any) ([]map[string]any, error) {
	return []map[string]any{}, nil
}

func mustInt64(v any) (int64, error) {
	switch n := v.(type) {
	case int64:
		return n, nil
	case int:
		return int64(n), nil
	case float64:
		return int64(n), nil
	case string:
		id, err := strconv.ParseInt(strings.TrimSpace(n), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid numeric value %q", n)
		}
		return id, nil
	default:
		return 0, fmt.Errorf("unsupported numeric type %T", v)
	}
}
