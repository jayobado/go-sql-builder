package sqb

import (
	"context"

	"github.com/jmoiron/sqlx"
)

func VacuumSQLite(ctx context.Context, d Dialect, db *sqlx.DB) error {
	if d.Name() != "sqlite" {
		return ErrNotSQLite
	}
	_, err := db.ExecContext(ctx, "VACUUM")
	return err
}
