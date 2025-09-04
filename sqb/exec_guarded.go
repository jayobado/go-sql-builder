package sqb

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type AuditMeta struct {
	Op     string
	Table  string
	Reason string
	SQL    string
	Args   []any
	DryRun bool
}

var GlobalAuditHook func(ctx context.Context, meta AuditMeta)

// ExecGuarded: runs in a tx, enforces maxRows (if >=0), optional dry-run, and audits.
func ExecGuarded(ctx context.Context, db *sqlx.DB, sqlText string, args []any, maxRows int64, dryRun bool, meta AuditMeta) (int64, error) {
	meta.SQL, meta.Args, meta.DryRun = sqlText, args, dryRun
	if GlobalAuditHook != nil {
		GlobalAuditHook(ctx, meta)
	}
	if dryRun {
		return 0, nil
	}
	tx, err := db.BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return 0, err
	}
	res, err := tx.ExecContext(ctx, sqlText, args...)
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	aff, err := res.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	if maxRows >= 0 && aff > maxRows {
		_ = tx.Rollback()
		return 0, fmt.Errorf("sqb: affected %d rows > max %d; rolled back", aff, maxRows)
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return aff, nil
}

func ExecGuardedBuilder(ctx context.Context, db *sqlx.DB, builder any, dryRun bool) (int64, error) {
	switch b := builder.(type) {
	case *UpdateBuilder:
		sqlText, args, err := b.Build()
		if err != nil {
			return 0, err
		}
		meta, max := b.guardMeta()
		return ExecGuarded(ctx, db, sqlText, args, max, dryRun, meta)
	case *DeleteBuilder:
		sqlText, args, err := b.Build()
		if err != nil {
			return 0, err
		}
		meta, max := b.guardMeta()
		return ExecGuarded(ctx, db, sqlText, args, max, dryRun, meta)
	case *TruncateBuilder:
		sqlText, args, err := b.Build()
		if err != nil {
			return 0, err
		}
		meta, max := b.guardMeta()
		return ExecGuarded(ctx, db, sqlText, args, max, dryRun, meta)
	default:
		return 0, fmt.Errorf("sqb: unsupported builder type %T", builder)
	}
}