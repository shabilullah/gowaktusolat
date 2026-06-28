package sqlite

import (
	"context"
	"fmt"

	"github.com/shabilullah/gowaktusolat/internal/repository"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

// ZoneRepo implements repository.ZoneRepository backed by SQLite.
type ZoneRepo struct {
	Pool *sqlitex.Pool
}

func (r *ZoneRepo) ListAll(ctx context.Context) ([]repository.ZoneRow, error) {
	conn, err := r.Pool.Take(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire db conn: %w", err)
	}
	defer r.Pool.Put(conn)

	var zones []repository.ZoneRow
	if err := sqlitex.ExecuteTransient(
		conn,
		"SELECT jakim_code, negeri, daerah FROM prayer_zones ORDER BY jakim_code",
		&sqlitex.ExecOptions{
			ResultFunc: func(stmt *sqlite.Stmt) error {
				zones = append(zones, repository.ZoneRow{
					JakimCode: stmt.ColumnText(0),
					Negeri:    stmt.ColumnText(1),
					Daerah:    stmt.ColumnText(2),
				})
				return nil
			},
		},
	); err != nil {
		return nil, fmt.Errorf("list zones: %w", err)
	}

	return zones, nil
}

func (r *ZoneRepo) ListByState(ctx context.Context, statePrefix string) ([]repository.ZoneRow, error) {
	conn, err := r.Pool.Take(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire db conn: %w", err)
	}
	defer r.Pool.Put(conn)

	var zones []repository.ZoneRow
	if err := sqlitex.ExecuteTransient(
		conn,
		"SELECT jakim_code, negeri, daerah FROM prayer_zones WHERE UPPER(jakim_code) LIKE ? ORDER BY jakim_code",
		&sqlitex.ExecOptions{
			Args: []interface{}{statePrefix + "%"},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				zones = append(zones, repository.ZoneRow{
					JakimCode: stmt.ColumnText(0),
					Negeri:    stmt.ColumnText(1),
					Daerah:    stmt.ColumnText(2),
				})
				return nil
			},
		},
	); err != nil {
		return nil, fmt.Errorf("list zones by state: %w", err)
	}

	return zones, nil
}

func (r *ZoneRepo) LookupDaerah(ctx context.Context, jakimCode string) (string, error) {
	conn, err := r.Pool.Take(ctx)
	if err != nil {
		return "", fmt.Errorf("acquire db conn: %w", err)
	}
	defer r.Pool.Put(conn)

	var daerah string
	if err := sqlitex.ExecuteTransient(conn,
		"SELECT daerah FROM prayer_zones WHERE jakim_code = ?",
		&sqlitex.ExecOptions{
			Args: []interface{}{jakimCode},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				daerah = stmt.ColumnText(0)
				return nil
			},
		}); err != nil {
		return "", fmt.Errorf("lookup daerah: %w", err)
	}
	return daerah, nil
}
