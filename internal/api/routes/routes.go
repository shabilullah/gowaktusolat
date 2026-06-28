package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/shabilullah/gowaktusolat/internal/service"
	"zombiezen.com/go/sqlite/sqlitex"
)

// RegisterAll sets up API middleware and registers every route group.
func RegisterAll(
	app *fiber.App,
	prayerSvc service.PrayerService,
	zoneSvc service.ZoneService,
	pdfSvc service.PDFService,
	lastUpdateDB *sqlitex.Pool,
	apiKey string,
) {
	apiGroup := app.Group("/api")

	setupMiddleware(app, apiGroup, apiKey)

	registerPrayerRoutes(apiGroup, prayerSvc)
	registerZoneRoutes(apiGroup, zoneSvc)
	registerJadualRoutes(apiGroup, prayerSvc, zoneSvc, pdfSvc)
	registerLastUpdateRoute(apiGroup, lastUpdateDB)
	registerCacheRoute(apiGroup, apiKey)
}
