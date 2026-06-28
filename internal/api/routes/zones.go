package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/shabilullah/gowaktusolat/internal/api"
	"github.com/shabilullah/gowaktusolat/internal/service"
)

func registerZoneRoutes(group fiber.Router, zoneSvc service.ZoneService) {
	h := &api.Zones{Service: zoneSvc}
	group.Get("/zones", h.Index)
	group.Get("/zones/:lat/:long", h.GetByCoordinate)
	group.Get("/zones/:state", h.GetByState)
}
