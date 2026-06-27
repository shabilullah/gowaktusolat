package api

import (
	"database/sql"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/shabilullah/gowaktusolat/internal/geo"
)

type Zones struct {
	DB       *sql.DB
	Detector *geo.Detector
}

func (h *Zones) Index(c fiber.Ctx) error {
	rows, err := h.DB.Query("SELECT jakim_code, negeri, daerah FROM prayer_zones ORDER BY jakim_code")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": err.Error()})
	}
	defer rows.Close()

	type zoneResult struct {
		JakimCode string `json:"jakimCode"`
		Negeri    string `json:"negeri"`
		Daerah    string `json:"daerah"`
	}

	var zones []zoneResult
	for rows.Next() {
		var z zoneResult
		if err := rows.Scan(&z.JakimCode, &z.Negeri, &z.Daerah); err != nil {
			return c.Status(500).JSON(fiber.Map{"message": err.Error()})
		}
		zones = append(zones, z)
	}
	if err := rows.Err(); err != nil {
		return c.Status(500).JSON(fiber.Map{"message": err.Error()})
	}

	return c.JSON(zones)
}

func (h *Zones) GetByState(c fiber.Ctx) error {
	state := strings.ToUpper(c.Params("state"))

	rows, err := h.DB.Query(
		"SELECT jakim_code, negeri, daerah FROM prayer_zones WHERE UPPER(jakim_code) LIKE ? ORDER BY jakim_code",
		state+"%",
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": err.Error()})
	}
	defer rows.Close()

	type zoneResult struct {
		JakimCode string `json:"jakimCode"`
		Negeri    string `json:"negeri"`
		Daerah    string `json:"daerah"`
	}

	var zones []zoneResult
	for rows.Next() {
		var z zoneResult
		if err := rows.Scan(&z.JakimCode, &z.Negeri, &z.Daerah); err != nil {
			return c.Status(500).JSON(fiber.Map{"message": err.Error()})
		}
		zones = append(zones, z)
	}
	if err := rows.Err(); err != nil {
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

func RegisterRoutes(app *fiber.App, database *sql.DB, detector *geo.Detector) {
	api := app.Group("/api")
	api.Use(func(c fiber.Ctx) error {
		c.Set("Cache-Control", "public, max-age=3600")
		return c.Next()
	})

	lastUpdateHandler := &LastUpdate{DB: database}
	api.Get("/last-update", lastUpdateHandler.Get)

	jadualHandler := &JadualSolat{DB: database}
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
	api.Post("/cache/reset", cacheHandler.Reset)

	app.Use(func(c fiber.Ctx) error {
		return c.Status(404).JSON(fiber.Map{
			"message": "No route matched. Please see the API documentation.",
		})
	})
}
