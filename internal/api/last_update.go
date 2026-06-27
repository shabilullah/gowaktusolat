package api

import (
	"github.com/gofiber/fiber/v3"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

type LastUpdate struct {
	DB *sqlitex.Pool
}

type lastUpdateResponse struct {
	LastRun    string `json:"last_run"`
	LastStatus string `json:"last_status"`
}

func (h *LastUpdate) Get(c fiber.Ctx) error {
	conn, err := h.DB.Take(c.Context())
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": err.Error()})
	}
	defer h.DB.Put(conn)
	var resp lastUpdateResponse

	if err := sqlitex.ExecuteTransient(
		conn,
		"SELECT value FROM settings WHERE key = 'scraper.last_run'",
		&sqlitex.ExecOptions{
			ResultFunc: func(stmt *sqlite.Stmt) error {
				resp.LastRun = stmt.ColumnText(0)
				return nil
			},
		},
	); err != nil {
		return c.Status(500).JSON(fiber.Map{"message": err.Error()})
	}

	if err := sqlitex.ExecuteTransient(
		conn,
		"SELECT value FROM settings WHERE key = 'scraper.last_status'",
		&sqlitex.ExecOptions{
			ResultFunc: func(stmt *sqlite.Stmt) error {
				resp.LastStatus = stmt.ColumnText(0)
				return nil
			},
		},
	); err != nil {
		return c.Status(500).JSON(fiber.Map{"message": err.Error()})
	}

	return c.JSON(resp)
}
