package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/shabilullah/gowaktusolat/internal/api"
	"github.com/shabilullah/gowaktusolat/internal/service"
)

func registerJadualRoutes(group fiber.Router, prayerSvc service.PrayerService, zoneSvc service.ZoneService, pdfSvc service.PDFService) {
	h := &api.JadualSolat{
		PrayerService: prayerSvc,
		ZoneService:   zoneSvc,
		PDFService:    pdfSvc,
	}
	group.Get("/jadual_solat/:zone", h.FetchMonth)
}
