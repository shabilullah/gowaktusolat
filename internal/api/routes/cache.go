package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/shabilullah/gowaktusolat/internal/api"
)

func registerCacheRoute(group fiber.Router, apiKey string) {
	h := &api.CacheHandler{}
	if apiKey != "" {
		group.Post("/cache/reset", keyauthMiddleware(apiKey), h.Reset)
	} else {
		group.Post("/cache/reset", h.Reset)
	}
}
