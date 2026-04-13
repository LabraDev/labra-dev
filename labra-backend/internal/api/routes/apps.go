package routes

import (
	"labra-backend/internal/api/handlers"

	"github.com/go-fuego/fuego"
)

func Apps(s *fuego.Server) {
	fuego.PostStd(s, "/v1/apps", handlers.CreateAppHandler)
	fuego.GetStd(s, "/v1/apps", handlers.ListAppsHandler)
	fuego.GetStd(s, "/v1/apps/{id}", handlers.GetAppHandler)
	fuego.PatchStd(s, "/v1/apps/{id}", handlers.PatchAppHandler)
	fuego.GetStd(s, "/v1/apps/{id}/env-vars", handlers.ListAppEnvVarsHandler)
	fuego.PostStd(s, "/v1/apps/{id}/env-vars", handlers.CreateAppEnvVarHandler)
	fuego.PatchStd(s, "/v1/apps/{id}/env-vars/{envID}", handlers.PatchAppEnvVarHandler)
	fuego.DeleteStd(s, "/v1/apps/{id}/env-vars/{envID}", handlers.DeleteAppEnvVarHandler)
	fuego.GetStd(s, "/v1/apps/{id}/releases", handlers.GetAppReleasesHandler)
	fuego.GetStd(s, "/v1/apps/{id}/rollbacks", handlers.GetAppRollbacksHandler)
	fuego.PostStd(s, "/v1/apps/{id}/rollback", handlers.CreateRollbackHandler)
	fuego.GetStd(s, "/v1/apps/{id}/health", handlers.GetAppHealthSummaryHandler)
	fuego.GetStd(s, "/v1/apps/{id}/observability", handlers.GetAppObservabilitySummaryHandler)
	fuego.GetStd(s, "/v1/apps/{id}/observability/log-query", handlers.QueryAppObservabilityLogsHandler)
}
