package api

import (
	"github.com/gofiber/fiber/v3"
	"github.com/shabilullah/gowaktusolat/internal/api/presenter"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

type LastUpdate struct {
	DB *sqlitex.Pool
}

func (h *LastUpdate) Get(c fiber.Ctx) error {
	conn, err := h.DB.Take(c.Context())
	if err != nil {
		return c.Status(500).JSON(presenter.Message(err.Error()))
	}
	defer h.DB.Put(conn)

	var resp presenter.LastUpdateResponse

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
		return c.Status(500).JSON(presenter.Message(err.Error()))
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
		return c.Status(500).JSON(presenter.Message(err.Error()))
	}

	return c.JSON(resp)
}
