package main

import (
	"context"
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // or mysql / mssql driver
	"github.com/you/sqb/sqb"
)

type User struct {
	ID     int64  `db:"id"`
	Email  string `db:"email"`
	Name   string `db:"name"`
	Status string `db:"status"`
}

func main() {
	ctx := context.Background()
	d := sqb.Postgres{} // swap to MySQL{}, sqb.SQLite{...}, sqb.SQLServer{}

	db, err := sqlx.Open("postgres", "postgres://user:pass@localhost:5432/app?sslmode=disable")
	if err != nil { log.Fatal(err) }
	defer db.Close()

	// SELECT
	sel, args, err := sqb.Select(d).
		Columns("id", "email", "name").
		From("users").
		Where(sqb.And(
			sqb.Eq("status", "active", d),
			sqb.In("id", []int{1,2,3}, d),
		)).
		OrderBy("id DESC").
		Limit(50).
		Build()
	if err != nil { log.Fatal(err) }
	var users []User
	if err := db.SelectContext(ctx, &users, sel, args...); err != nil { log.Fatal(err) }

	// Multi-row INSERT + Postgres upsert
	ins := sqb.Insert(d).
		Into("users").
		Columns("email", "name", "status").
		ValuesRows([][]any{
			{"alice@example.com","Alice","active"},
			{"bob@example.com","Bob","active"},
		}).
		OnConflictDoUpdate(
			sqb.ConflictColumns("email"),
			sqb.MergeSets(
				sqb.SetExcluded(d, "name", "status"),
				map[string]sqb.Expr{"updated_at": sqb.Raw("NOW()")},
			),
		).
		Returning("id")
	isql, iargs, err := ins.Build()
	if err != nil { log.Fatal(err) }
	rows, err := db.QueryxContext(ctx, isql, iargs...)
	if err != nil { log.Fatal(err) }
	defer rows.Close()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil { log.Fatal(err) }
	}

	// UPDATE with default RequireWhere guard
	up := sqb.Update(d).
		Table("users").
		Set("status", "disabled").
		Where(sqb.Eq("id", 1, d))
	usql, uargs, err := up.Build()
	if err != nil { log.Fatal(err) }
	if _, err := db.ExecContext(ctx, usql, uargs...); err != nil { log.Fatal(err) }

	// Intentional table-wide DELETE with cap + audit
	sqb.GlobalAuditHook = func(ctx context.Context, m sqb.AuditMeta) {
		log.Printf("[AUDIT] %s %s reason=%q dry=%v sql=%s args=%v",
			m.Op, m.Table, m.Reason, m.DryRun, m.SQL, m.Args)
	}
	_, err = sqb.ExecGuardedBuilder(ctx, db,
		sqb.Delete(d).From("tmp_events").
			AllowFullTableWithMax(10_000_000).
			Reason("cleanup old staging"),
		false,
	)
	if err != nil { log.Fatal(err) }
}
