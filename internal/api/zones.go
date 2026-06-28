package api

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/shabilullah/gowaktusolat/internal/api/presenter"
	"github.com/shabilullah/gowaktusolat/internal/service"
)

type Zones struct {
	Service service.ZoneService
}

func (h *Zones) Index(c fiber.Ctx) error {
	zones, err := h.Service.ListAll(c.Context())
	if err != nil {
		return c.Status(500).JSON(presenter.Message(err.Error()))
	}

	items := make([]presenter.ZoneItem, len(zones))
	for i, z := range zones {
		items[i] = presenter.ZoneItem{
			JakimCode: z.JakimCode,
			Negeri:    z.Negeri,
			Daerah:    z.Daerah,
		}
	}
	return c.JSON(items)
}

func (h *Zones) GetByState(c fiber.Ctx) error {
	state := strings.ToUpper(c.Params("state"))

	zones, err := h.Service.ListByState(c.Context(), state)
	if err != nil {
		return c.Status(500).JSON(presenter.Message(err.Error()))
	}

	items := make([]presenter.ZoneItem, len(zones))
	for i, z := range zones {
		items[i] = presenter.ZoneItem{
			JakimCode: z.JakimCode,
			Negeri:    z.Negeri,
			Daerah:    z.Daerah,
		}
	}
	return c.JSON(items)
}

func (h *Zones) GetByCoordinate(c fiber.Ctx) error {
	latStr := c.Params("lat")
	lngStr := c.Params("long")

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		return c.Status(422).JSON(presenter.Message("Invalid latitude"))
	}
	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil {
		return c.Status(422).JSON(presenter.Message("Invalid longitude"))
	}

	zone, err := h.Service.GetByCoordinate(c.Context(), lat, lng)
	if err != nil {
		return c.Status(404).JSON(presenter.Message(err.Error()))
	}

	return c.JSON(presenter.ZoneByCoordinateResponse{
		Zone:     zone.JakimCode,
		State:    zone.Negeri,
		District: zone.Daerah,
	})
}
