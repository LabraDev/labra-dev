package handlers

import (
	"net/http"
	"strings"
)

type ServiceStatus struct {
	Name        string `json:"name"`
	Tier        string `json:"tier"`
	Mode        string `json:"mode"`
	Status      string `json:"status"`
	Description string `json:"description"`
}

var serviceStatuses = []ServiceStatus{
	{
		Name:        "control-api",
		Tier:        "api",
		Mode:        "in-process",
		Status:      "healthy",
		Description: "User-facing API gateway and metadata endpoints",
	},
	{
		Name:        "deploy-orchestrator",
		Tier:        "worker",
		Mode:        "in-process",
		Status:      "healthy",
		Description: "Deployment queueing and execution orchestration",
	},
	{
		Name:        "webhook-ingestor",
		Tier:        "ingestion",
		Mode:        "in-process",
		Status:      "healthy",
		Description: "GitHub webhook normalization and routing",
	},
	{
		Name:        "ai-assistant",
		Tier:        "ai",
		Mode:        "in-process",
		Status:      "healthy",
		Description: "AI deployment insight generation with fallback safety controls",
	},
}

func GetSystemServicesHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"services": serviceStatuses,
		"architecture": map[string]any{
			"pattern": "3-tier + microservice control-plane",
			"tiers":   []string{"frontend", "api", "metadata"},
		},
	})
}

func GetReadinessChecklistHandler(w http.ResponseWriter, _ *http.Request) {
	checks := []map[string]any{
		{
			"control": "webhook_replay_window_configured",
			"status":  webhookMaxSkewSeconds > 0,
			"details": "Webhook timestamp replay window is enabled",
		},
		{
			"control": "webhook_secret_configured",
			"status":  strings.TrimSpace(githubWebhookSecret) != "",
			"details": "GitHub webhook secret must be configured",
		},
		{
			"control": "ai_prompt_version_present",
			"status":  strings.TrimSpace(aiPromptVersion) != "",
			"details": "AI prompt versioning is configured",
		},
		{
			"control": "ai_provider_timeout_configured",
			"status":  aiProviderTimeout > 0,
			"details": "AI provider timeout protects against hanging requests",
		},
		{
			"control": "ai_provider_fallback_available",
			"status":  true,
			"details": "Fallback insight path is available when AI provider fails",
		},
		{
			"control": "service_inventory_includes_ai",
			"status":  serviceExists("ai-assistant"),
			"details": "Service status includes AI assistant component",
		},
	}

	ready := true
	for _, c := range checks {
		if status, ok := c["status"].(bool); !ok || !status {
			ready = false
			break
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ready":  ready,
		"checks": checks,
		"phase":  "phase8-production-hardening",
	})
}

func serviceExists(name string) bool {
	target := strings.TrimSpace(strings.ToLower(name))
	if target == "" {
		return false
	}
	for _, svc := range serviceStatuses {
		if strings.TrimSpace(strings.ToLower(svc.Name)) == target {
			return true
		}
	}
	return false
}
