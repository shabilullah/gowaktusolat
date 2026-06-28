package api

import (
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/extractors"
	"github.com/gofiber/fiber/v3/middleware/cache"
	"github.com/gofiber/fiber/v3/middleware/keyauth"

	"github.com/shabilullah/gowaktusolat/internal/api/presenter"
	"github.com/shabilullah/gowaktusolat/internal/geo"
	reposqlite "github.com/shabilullah/gowaktusolat/internal/repository/sqlite"
)

type Zones struct {
	ZoneRepo *reposqlite.ZoneRepo
	Detector *geo.Detector
}

func (h *Zones) Index(c fiber.Ctx) error {
	zones, err := h.ZoneRepo.ListAll(c.Context())
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

	zones, err := h.ZoneRepo.ListByState(c.Context(), state)
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

	result, err := h.Detector.DetectZone(lat, lng)
	if err != nil {
		return c.Status(404).JSON(presenter.Message(err.Error()))
	}

	return c.JSON(presenter.ZoneByCoordinateResponse{
		Zone:     result.Zone,
		State:    result.State,
		District: result.District,
	})
}

func RegisterRoutes(app *fiber.App, prayerRepo *reposqlite.PrayerTimeRepo, zoneRepo *reposqlite.ZoneRepo, detector *geo.Detector, apiKey string) {
	configuredAPIKey = apiKey

	api := app.Group("/api")

	api.Use(func(c fiber.Ctx) error {
		c.Set("Cache-Control", "public, max-age=3600")
		return c.Next()
	})

	api.Use(func(c fiber.Ctx) error {
		if configuredAPIKey != "" && fiber.Query[bool](c, "invalidateCache") {
			if c.Get("X-API-Key") != configuredAPIKey {
				return c.Status(fiber.StatusUnauthorized).JSON(presenter.Message("unauthorized"))
			}
		}
		return c.Next()
	})

	api.Use(cache.New(cache.Config{
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

	lastUpdateHandler := &LastUpdate{DB: prayerRepo.Pool}
	api.Get("/last-update", lastUpdateHandler.Get)

	jadualHandler := &JadualSolat{PrayerRepo: prayerRepo, ZoneRepo: zoneRepo}
	api.Get("/jadual_solat/:zone", jadualHandler.FetchMonth)

	prayerHandler := &PrayerTime{Repo: prayerRepo, Detector: detector}
	api.Get("/solat/:zone", prayerHandler.FetchMonth)
	api.Get("/solat/:zone/:day", prayerHandler.FetchDay)
	api.Get("/solat/gps/:lat/:long", prayerHandler.FetchMonthByGPS)

	zonesHandler := &Zones{ZoneRepo: zoneRepo, Detector: detector}
	api.Get("/zones", zonesHandler.Index)
	api.Get("/zones/:lat/:long", zonesHandler.GetByCoordinate)
	api.Get("/zones/:state", zonesHandler.GetByState)

	cacheHandler := &CacheHandler{}
	if apiKey != "" {
		api.Post("/cache/reset", keyauthMiddleware(apiKey), cacheHandler.Reset)
	} else {
		api.Post("/cache/reset", cacheHandler.Reset)
	}

	app.Use(func(c fiber.Ctx) error {
		return c.Status(404).JSON(presenter.Message("No route matched. Please see the API documentation."))
	})
}

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
