package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/shabilullah/gowaktusolat/internal/api"
	"github.com/shabilullah/gowaktusolat/internal/service"
)

func registerPrayerRoutes(group fiber.Router, prayerSvc service.PrayerService) {
	h := &api.PrayerTime{Service: prayerSvc}
	group.Get("/solat/:zone", h.FetchMonth)
	group.Get("/solat/:zone/:day", h.FetchDay)
	group.Get("/solat/gps/:lat/:long", h.FetchMonthByGPS)
}
