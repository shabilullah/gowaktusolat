package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/shabilullah/gowaktusolat/internal/api"
	"zombiezen.com/go/sqlite/sqlitex"
)

func registerLastUpdateRoute(group fiber.Router, db *sqlitex.Pool) {
	h := &api.LastUpdate{DB: db}
	group.Get("/last-update", h.Get)
}
