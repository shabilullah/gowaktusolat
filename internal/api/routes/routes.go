package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/shabilullah/gowaktusolat/internal/api/presenter"
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

	setupAPIMiddleware(apiGroup, apiKey)

	registerPrayerRoutes(apiGroup, prayerSvc)
	registerZoneRoutes(apiGroup, zoneSvc)
	registerJadualRoutes(apiGroup, prayerSvc, zoneSvc, pdfSvc)
	registerLastUpdateRoute(apiGroup, lastUpdateDB)
	registerCacheRoute(apiGroup, apiKey)

	// 404 catch-all — must be registered after all routes.
	app.Use(func(c fiber.Ctx) error {
		return c.Status(404).JSON(presenter.Message("No route matched. Please see the API documentation."))
	})
}
