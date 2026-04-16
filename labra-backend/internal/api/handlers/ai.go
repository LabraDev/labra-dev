package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"labra-backend/internal/api/store"
)

type AIRuntimeConfig struct {
	FeatureEnabled  bool
	KillSwitch      bool
	PromptVersion   string
	ProviderModel   string
	ProviderTimeout time.Duration
	ProviderRetries int
	Provider        AIProvider
}

type AIProviderInput struct {
	Prompt        string
	PromptVersion string
	Model         string
}

type AIProviderOutput struct {
	Text       string
	Provider   string
	Model      string
	Confidence string
}

type AIProvider interface {
	Generate(ctx context.Context, in AIProviderInput) (AIProviderOutput, error)
}

type localMCPProvider struct{}

func (p localMCPProvider) Generate(_ context.Context, in AIProviderInput) (AIProviderOutput, error) {
	prompt := strings.TrimSpace(in.Prompt)
	if prompt == "" {
		return AIProviderOutput{}, fmt.Errorf("empty prompt")
	}

	text := "AI insight: deployment health looks stable. Next step is to verify logs for warnings and validate post-deploy smoke routes."
	if strings.Contains(strings.ToLower(prompt), "failed") {
		text = "AI insight: deployment appears unhealthy. Prioritize first error log line, rerun build locally, then retry deployment after fixing root cause."
	}

	return AIProviderOutput{
		Text:       text,
		Provider:   "mcp-simulated",
		Model:      strings.TrimSpace(in.Model),
		Confidence: "medium",
	}, nil
}

var (
	aiFeatureEnabled             = true
	aiKillSwitch                 = false
	aiPromptVersion              = "phase7-v1"
	aiProviderModel              = "mock-ops-v1"
	aiProviderTimeout            = 1800 * time.Millisecond
	aiProviderRetries            = 2
	aiProvider        AIProvider = localMCPProvider{}
)

var (
	emailRegex    = regexp.MustCompile(`(?i)\b[A-Z0-9._%+\-]+@[A-Z0-9.\-]+\.[A-Z]{2,}\b`)
	awsAKRegex    = regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`)
	secretKVRegex = regexp.MustCompile(`(?i)(aws_secret_access_key|api[_-]?key|token|password)\s*[:=]\s*([^\s,;]+)`)
	bearerRegex   = regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9\-._~+/]+=*`)
)

type aiDeployInsightRequest struct {
	DeploymentID int64  `json:"deployment_id"`
	Prompt       string `json:"prompt"`
	BypassAI     bool   `json:"bypass_ai"`
}

func InitAIRuntime(cfg AIRuntimeConfig) {
	aiFeatureEnabled = cfg.FeatureEnabled
	aiKillSwitch = cfg.KillSwitch

	if strings.TrimSpace(cfg.PromptVersion) != "" {
		aiPromptVersion = strings.TrimSpace(cfg.PromptVersion)
	}
	if strings.TrimSpace(cfg.ProviderModel) != "" {
		aiProviderModel = strings.TrimSpace(cfg.ProviderModel)
	}
	if cfg.ProviderTimeout > 0 {
		aiProviderTimeout = cfg.ProviderTimeout
	}
	if cfg.ProviderRetries >= 0 {
		aiProviderRetries = cfg.ProviderRetries
	}
	if cfg.Provider != nil {
		aiProvider = cfg.Provider
	}
}

func PostAIDeployInsightsHandler(w http.ResponseWriter, r *http.Request) {
	if appStore == nil {
		writeJSONError(w, http.StatusInternalServerError, "store not initialized")
		return
	}

	userID, ok := readUserID(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing user id: pass X-User-ID header")
		return
	}

	var body aiDeployInsightRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if body.DeploymentID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "deployment_id must be a positive integer")
		return
	}

	deployment, err := appStore.GetDeploymentByIDForUser(r.Context(), body.DeploymentID, userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "deployment not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to load deployment")
		return
	}

	logs, err := appStore.ListDeploymentLogs(r.Context(), deployment.ID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to load deployment logs")
		return
	}

	rawPrompt := buildDeployPrompt(deployment, logs, strings.TrimSpace(body.Prompt))
	redactedPrompt, inputRedacted := redactSensitive(rawPrompt)

	resultText := ""
	resultProvider := "fallback"
	resultModel := "n/a"
	confidence := "low"
	fallbackUsed := true
	status := "fallback"
	failureReason := ""

	switch {
	case body.BypassAI:
		resultText = fallbackInsight(deployment, logs, "bypass requested")
		status = "bypass"
	case !aiFeatureEnabled || aiKillSwitch:
		reason := "AI feature is disabled"
		if aiKillSwitch {
			reason = "AI kill switch enabled"
		}
		resultText = fallbackInsight(deployment, logs, reason)
		status = "disabled_fallback"
	default:
		out, aiErr := invokeAIWithRetries(r.Context(), AIProviderInput{
			Prompt:        redactedPrompt,
			PromptVersion: aiPromptVersion,
			Model:         aiProviderModel,
		})
		if aiErr != nil {
			failureReason = aiErr.Error()
			resultText = fallbackInsight(deployment, logs, "AI provider unavailable")
			status = "fallback"
		} else {
			resultText = strings.TrimSpace(out.Text)
			resultProvider = strings.TrimSpace(out.Provider)
			if resultProvider == "" {
				resultProvider = "provider"
			}
			resultModel = strings.TrimSpace(out.Model)
			if resultModel == "" {
				resultModel = aiProviderModel
			}
			confidence = strings.TrimSpace(out.Confidence)
			if confidence == "" {
				confidence = "medium"
			}
			fallbackUsed = false
			status = "succeeded"
		}
	}

	resultText = limitLen(strings.TrimSpace(resultText), 1500)
	if resultText == "" {
		resultText = fallbackInsight(deployment, logs, "empty AI response")
		fallbackUsed = true
		status = "fallback"
		resultProvider = "fallback"
		resultModel = "n/a"
		confidence = "low"
	}

	logEntry, logErr := appStore.CreateAIRequestLog(r.Context(), store.CreateAIRequestLogInput{
		UserID:        userID,
		DeploymentID:  deployment.ID,
		PromptVersion: aiPromptVersion,
		Provider:      resultProvider,
		Model:         resultModel,
		InputRedacted: inputRedacted,
		FallbackUsed:  fallbackUsed,
		Status:        status,
		InputExcerpt:  limitLen(redactedPrompt, 500),
		OutputExcerpt: limitLen(resultText, 500),
	})
	if logErr != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to persist ai request log")
		return
	}

	auditMeta, _ := json.Marshal(map[string]any{
		"deployment_id": deployment.ID,
		"provider":      resultProvider,
		"model":         resultModel,
		"fallback_used": fallbackUsed,
		"status":        status,
	})
	_ = appStore.CreateAuditEvent(r.Context(), store.AuditEventInput{
		ActorUserID: userID,
		EventType:   "ai.deploy_insight",
		TargetType:  "deployment",
		TargetID:    strconv.FormatInt(deployment.ID, 10),
		Status:      status,
		Message:     strings.TrimSpace(failureReason),
		Metadata:    string(auditMeta),
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"deployment_id":  deployment.ID,
		"insight":        resultText,
		"source":         resultProvider,
		"model":          resultModel,
		"prompt_version": aiPromptVersion,
		"fallback_used":  fallbackUsed,
		"confidence":     confidence,
		"limitations":    "AI-generated output may be incomplete or incorrect. Verify against deployment logs.",
		"request_log":    logEntry,
	})
}

func GetAIRequestLogsHandler(w http.ResponseWriter, r *http.Request) {
	if appStore == nil {
		writeJSONError(w, http.StatusInternalServerError, "store not initialized")
		return
	}

	userID, ok := readUserID(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing user id: pass X-User-ID header")
		return
	}

	limit := 20
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			writeJSONError(w, http.StatusBadRequest, "limit must be a positive integer")
			return
		}
		limit = parsed
	}

	logs, err := appStore.ListAIRequestLogsByUser(r.Context(), userID, limit)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to load ai request logs")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"logs":  logs,
		"count": len(logs),
	})
}

func invokeAIWithRetries(ctx context.Context, in AIProviderInput) (AIProviderOutput, error) {
	if aiProvider == nil {
		return AIProviderOutput{}, fmt.Errorf("ai provider is not configured")
	}

	attempts := aiProviderRetries + 1
	if attempts < 1 {
		attempts = 1
	}

	var lastErr error
	for i := 0; i < attempts; i++ {
		runCtx, cancel := context.WithTimeout(ctx, aiProviderTimeout)
		out, err := aiProvider.Generate(runCtx, in)
		cancel()
		if err == nil {
			if strings.TrimSpace(out.Text) == "" {
				lastErr = fmt.Errorf("ai provider returned empty output")
				continue
			}
			return out, nil
		}
		lastErr = err
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("ai provider failed")
	}
	return AIProviderOutput{}, lastErr
}

func buildDeployPrompt(dep store.Deployment, logs []store.DeploymentLog, userPrompt string) string {
	parts := []string{
		"You are a deployment assistant for Labra.",
		fmt.Sprintf("Deployment id: %d", dep.ID),
		fmt.Sprintf("Status: %s", dep.Status),
		fmt.Sprintf("Trigger: %s", dep.TriggerType),
		fmt.Sprintf("Branch: %s", dep.Branch),
		fmt.Sprintf("Failure reason: %s", dep.FailureReason),
	}
	if dep.CommitSHA != "" {
		parts = append(parts, fmt.Sprintf("Commit: %s", dep.CommitSHA))
	}

	maxLogs := len(logs)
	if maxLogs > 10 {
		maxLogs = 10
	}
	for i := 0; i < maxLogs; i++ {
		msg := strings.TrimSpace(logs[i].Message)
		if msg == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("Log %d (%s): %s", i+1, logs[i].LogLevel, limitLen(msg, 180)))
	}

	userPrompt = strings.TrimSpace(userPrompt)
	if userPrompt != "" {
		parts = append(parts, "User ask: "+limitLen(userPrompt, 500))
	}

	return strings.Join(parts, "\n")
}

func fallbackInsight(dep store.Deployment, logs []store.DeploymentLog, reason string) string {
	status := strings.TrimSpace(dep.Status)
	if status == "" {
		status = "unknown"
	}
	base := fmt.Sprintf("Fallback insight (%s): deployment status is %s.", strings.TrimSpace(reason), status)
	if len(logs) > 0 {
		last := strings.TrimSpace(logs[len(logs)-1].Message)
		if last != "" {
			return base + " Latest log: " + limitLen(last, 180)
		}
	}
	if strings.TrimSpace(dep.FailureReason) != "" {
		return base + " Failure reason: " + limitLen(dep.FailureReason, 180)
	}
	return base + " Review deployment logs for detailed diagnostics."
}

func redactSensitive(in string) (string, bool) {
	out := in
	out = emailRegex.ReplaceAllString(out, "[REDACTED_EMAIL]")
	out = awsAKRegex.ReplaceAllString(out, "[REDACTED_AWS_ACCESS_KEY]")
	out = bearerRegex.ReplaceAllString(out, "Bearer [REDACTED_TOKEN]")
	out = secretKVRegex.ReplaceAllString(out, "$1=[REDACTED]")
	return out, out != in
}

func limitLen(in string, max int) string {
	if max <= 0 {
		return ""
	}
	v := strings.TrimSpace(in)
	if len(v) <= max {
		return v
	}
	if max <= 3 {
		return v[:max]
	}
	return v[:max-3] + "..."
}
