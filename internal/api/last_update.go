package api

import (
	"database/sql"

	"github.com/gofiber/fiber/v3"
)

type LastUpdate struct {
	DB *sql.DB
}

type lastUpdateResponse struct {
	LastRun    string `json:"last_run"`
	LastStatus string `json:"last_status"`
}

func (h *LastUpdate) Get(c fiber.Ctx) error {
	var resp lastUpdateResponse

	if err := h.DB.QueryRow(
		"SELECT value FROM settings WHERE key = 'scraper.last_run'",
	).Scan(&resp.LastRun); err != nil && err != sql.ErrNoRows {
		return c.Status(500).JSON(fiber.Map{"message": err.Error()})
	}

	if err := h.DB.QueryRow(
		"SELECT value FROM settings WHERE key = 'scraper.last_status'",
	).Scan(&resp.LastStatus); err != nil && err != sql.ErrNoRows {
		return c.Status(500).JSON(fiber.Map{"message": err.Error()})
	}

	return c.JSON(resp)
}
