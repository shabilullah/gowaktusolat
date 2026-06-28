package routes

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/extractors"
	"github.com/gofiber/fiber/v3/middleware/cache"
	"github.com/gofiber/fiber/v3/middleware/keyauth"

	"github.com/shabilullah/gowaktusolat/internal/api/presenter"
)

var configuredAPIKey string

func keyauthMiddleware(key string) fiber.Handler {
	return keyauth.New(keyauth.Config{
		Extractor: extractors.FromHeader("X-API-Key"),
		Validator: func(c fiber.Ctx, k string) (bool, error) {
			return k == key, nil
		},
		ErrorHandler: func(c fiber.Ctx, err error) error {
			return c.Status(fiber.StatusUnauthorized).JSON(presenter.Message("unauthorized"))
		},
	})
}

func setupAPIMiddleware(group fiber.Router, apiKey string) {
	configuredAPIKey = apiKey

	group.Use(func(c fiber.Ctx) error {
		c.Set("Cache-Control", "public, max-age=3600")
		return c.Next()
	})

	group.Use(func(c fiber.Ctx) error {
		if configuredAPIKey != "" && fiber.Query[bool](c, "invalidateCache") {
			if c.Get("X-API-Key") != configuredAPIKey {
				return c.Status(fiber.StatusUnauthorized).JSON(presenter.Message("unauthorized"))
			}
		}
		return c.Next()
	})

	group.Use(cache.New(cache.Config{
		Next: func(c fiber.Ctx) bool {
			path := c.Path()
			if strings.Contains(path, "jadual_solat") || strings.Contains(path, "cache/reset") {
				return true
			}
			return false
		},
		ExpirationGenerator: func(c fiber.Ctx, cfg *cache.Config) time.Duration {
			path := c.Path()
			switch {
			case strings.Contains(path, "/zones"):
				return 6 * time.Hour
			case strings.Contains(path, "/solat"):
				return 1 * time.Hour
			default:
				return 5 * time.Minute
			}
		},
		CacheInvalidator: func(c fiber.Ctx) bool {
			return fiber.Query[bool](c, "invalidateCache")
		},
	}))
}
