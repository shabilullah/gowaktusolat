package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/robfig/cron/v3"
)

type Settings struct {
	DB *sql.DB
}

type scraperSettings struct {
	Enabled    bool   `json:"enabled"`
	Schedule   string `json:"schedule"`
	LastRun    string `json:"last_run"`
	LastStatus string `json:"last_status"`
}

type settingsResponse struct {
	Scraper scraperSettings `json:"scraper"`
}

type settingsPutRequest struct {
	Scraper *struct {
		Enabled  *bool   `json:"enabled"`
		Schedule *string `json:"schedule"`
	} `json:"scraper"`
}

func (h *Settings) Get(c fiber.Ctx) error {
	resp := settingsResponse{}

	if err := h.loadScraperSettings(&resp.Scraper); err != nil {
		return c.Status(500).JSON(fiber.Map{"message": err.Error()})
	}

	return c.JSON(resp)
}

func (h *Settings) Put(c fiber.Ctx) error {
	body := c.Body()

	var req settingsPutRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid JSON"})
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err == nil {
		for key := range raw {
			if key != "scraper" {
				return c.Status(400).JSON(fiber.Map{"message": fmt.Sprintf("Unknown key: %s", key)})
			}
		}
	}

	if req.Scraper != nil {
		if req.Scraper.Schedule != nil {
			parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
			if _, err := parser.Parse(*req.Scraper.Schedule); err != nil {
				return c.Status(400).JSON(fiber.Map{"message": "Invalid cron expression"})
			}
			if _, err := h.DB.Exec(
				"INSERT OR REPLACE INTO settings (key, value, updated_at) VALUES ('scraper.schedule', ?, datetime('now'))",
				*req.Scraper.Schedule,
			); err != nil {
				return c.Status(500).JSON(fiber.Map{"message": err.Error()})
			}
		}

		if req.Scraper.Enabled != nil {
			enabled := strconv.FormatBool(*req.Scraper.Enabled)
			if _, err := h.DB.Exec(
				"INSERT OR REPLACE INTO settings (key, value, updated_at) VALUES ('scraper.enabled', ?, datetime('now'))",
				enabled,
			); err != nil {
				return c.Status(500).JSON(fiber.Map{"message": err.Error()})
			}
		}

		var scraperRaw map[string]json.RawMessage
		scraperBytes, _ := json.Marshal(req.Scraper)
		if err := json.Unmarshal(scraperBytes, &scraperRaw); err == nil {
			knownKeys := map[string]bool{"enabled": true, "schedule": true, "last_run": true, "last_status": true}
			for key := range scraperRaw {
				if !knownKeys[key] {
					return c.Status(400).JSON(fiber.Map{"message": fmt.Sprintf("Unknown scraper key: %s", key)})
				}
			}
		}
	}

	resp := settingsResponse{}
	if err := h.loadScraperSettings(&resp.Scraper); err != nil {
		return c.Status(500).JSON(fiber.Map{"message": err.Error()})
	}
	return c.JSON(resp)
}

func (h *Settings) loadScraperSettings(s *scraperSettings) error {
	rows, err := h.DB.Query("SELECT key, value FROM settings WHERE key LIKE 'scraper.%'")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return err
		}
		switch {
		case strings.HasSuffix(key, ".enabled"):
			s.Enabled = value == "true"
		case strings.HasSuffix(key, ".schedule"):
			s.Schedule = value
		case strings.HasSuffix(key, ".last_run"):
			s.LastRun = value
		case strings.HasSuffix(key, ".last_status"):
			s.LastStatus = value
		}
	}
	return rows.Err()
}
