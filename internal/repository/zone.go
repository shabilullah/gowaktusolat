package repository

import "context"

// ZoneRow is a single zone record from the database.
type ZoneRow struct {
	JakimCode string
	Negeri    string
	Daerah    string
}

// ZoneRepository defines the data-access contract for prayer zones.
type ZoneRepository interface {
	ListAll(ctx context.Context) ([]ZoneRow, error)
	ListByState(ctx context.Context, statePrefix string) ([]ZoneRow, error)
	LookupDaerah(ctx context.Context, jakimCode string) (string, error)
}
