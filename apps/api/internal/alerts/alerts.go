package alerts

import "github.com/jackc/pgx/v5/pgxpool"

// Dispatcher checks gauge readings against user-defined thresholds and
// dispatches alerts via SMS, push, and Discord webhooks.
// TODO: implement in Phase 3 alongside user accounts and alert preferences.
type Dispatcher struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Dispatcher {
	return &Dispatcher{db: db}
}
