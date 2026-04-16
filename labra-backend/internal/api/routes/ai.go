package routes

import (
	"labra-backend/internal/api/handlers"

	"github.com/go-fuego/fuego"
)

func AIRoutes(s *fuego.Server) {
	fuego.PostStd(s, "/v1/ai/deploy-insights", withAuth(handlers.PostAIDeployInsightsHandler))
	fuego.GetStd(s, "/v1/ai/requests", withAuth(handlers.GetAIRequestLogsHandler))
}
