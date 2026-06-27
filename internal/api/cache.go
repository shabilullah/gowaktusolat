package api

import (
	"github.com/gofiber/fiber/v3"
)

type CacheHandler struct{}

func (h *CacheHandler) Reset(c fiber.Ctx) error {
	c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	return c.JSON(fiber.Map{
		"message": "Cache invalidated",
	})
}
