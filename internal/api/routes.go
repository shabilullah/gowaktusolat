package api

import (
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/extractors"
	"github.com/gofiber/fiber/v3/middleware/cache"
	"github.com/gofiber/fiber/v3/middleware/keyauth"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"

	"github.com/shabilullah/gowaktusolat/internal/geo"
)

type Zones struct {
	DB       *sqlitex.Pool
	Detector *geo.Detector
}

func (h *Zones) Index(c fiber.Ctx) error {
	conn, err := h.DB.Take(c.Context())
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": err.Error()})
	}
	defer h.DB.Put(conn)
	type zoneResult struct {
		JakimCode string `json:"jakimCode"`
		Negeri    string `json:"negeri"`
		Daerah    string `json:"daerah"`
	}

	var zones []zoneResult
	if err := sqlitex.ExecuteTransient(
		conn,
		"SELECT jakim_code, negeri, daerah FROM prayer_zones ORDER BY jakim_code",
		&sqlitex.ExecOptions{
			ResultFunc: func(stmt *sqlite.Stmt) error {
				zones = append(zones, zoneResult{
					JakimCode: stmt.ColumnText(0),
					Negeri:    stmt.ColumnText(1),
					Daerah:    stmt.ColumnText(2),
				})
				return nil
			},
		},
	); err != nil {
		return c.Status(500).JSON(fiber.Map{"message": err.Error()})
	}

	return c.JSON(zones)
}

func (h *Zones) GetByState(c fiber.Ctx) error {
	conn, err := h.DB.Take(c.Context())
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": err.Error()})
	}
	defer h.DB.Put(conn)
	state := strings.ToUpper(c.Params("state"))

	type zoneResult struct {
		JakimCode string `json:"jakimCode"`
		Negeri    string `json:"negeri"`
		Daerah    string `json:"daerah"`
	}

	var zones []zoneResult
	if err := sqlitex.ExecuteTransient(
		conn,
		"SELECT jakim_code, negeri, daerah FROM prayer_zones WHERE UPPER(jakim_code) LIKE ? ORDER BY jakim_code",
		&sqlitex.ExecOptions{
			Args: []interface{}{state + "%"},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				zones = append(zones, zoneResult{
					JakimCode: stmt.ColumnText(0),
					Negeri:    stmt.ColumnText(1),
					Daerah:    stmt.ColumnText(2),
				})
				return nil
			},
		},
	); err != nil {
		return c.Status(500).JSON(fiber.Map{"message": err.Error()})
	}

	return c.JSON(zones)
}

func (h *Zones) GetByCoordinate(c fiber.Ctx) error {
	latStr := c.Params("lat")
	lngStr := c.Params("long")

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		return c.Status(422).JSON(fiber.Map{"message": "Invalid latitude"})
	}
	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil {
		return c.Status(422).JSON(fiber.Map{"message": "Invalid longitude"})
	}

	result, err := h.Detector.DetectZone(lat, lng)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"message": err.Error()})
	}

	return c.JSON(fiber.Map{
		"zone":     result.Zone,
		"state":    result.State,
		"district": result.District,
	})
}

func RegisterRoutes(app *fiber.App, database *sqlitex.Pool, detector *geo.Detector, apiKey string) {
	configuredAPIKey = apiKey

	api := app.Group("/api")

	api.Use(func(c fiber.Ctx) error {
		c.Set("Cache-Control", "public, max-age=3600")
		return c.Next()
	})

	// Protect ?invalidateCache=true with API key when configured
	api.Use(func(c fiber.Ctx) error {
		if configuredAPIKey != "" && fiber.Query[bool](c, "invalidateCache") {
			if c.Get("X-API-Key") != configuredAPIKey {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"message": "unauthorized",
				})
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

	lastUpdateHandler := &LastUpdate{DB: database}
	api.Get("/last-update", lastUpdateHandler.Get)

	jadualHandler := &JadualSolat{Pool: database}
	api.Get("/jadual_solat/:zone", jadualHandler.FetchMonth)

	prayerHandler := &PrayerTime{DB: database, Detector: detector}
	api.Get("/solat/:zone", prayerHandler.FetchMonth)
	api.Get("/solat/:zone/:day", prayerHandler.FetchDay)
	api.Get("/solat/gps/:lat/:long", prayerHandler.FetchMonthByGPS)

	zonesHandler := &Zones{DB: database, Detector: detector}
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
		return c.Status(404).JSON(fiber.Map{
			"message": "No route matched. Please see the API documentation.",
		})
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
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "unauthorized",
			})
		},
	})
}
