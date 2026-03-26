package main

import (
	"fmt"

	"github.com/jayobado/go-sql-builder/sqb"
)

// --- helpers ---

func printHeader(title string) {
	fmt.Println()
	fmt.Println("------------------------------------------------------------")
	fmt.Println(title)
	fmt.Println("------------------------------------------------------------")
}

func printSQL(sql string, args []any) {
	fmt.Println(sql)
	if len(args) > 0 {
		fmt.Printf("ARGS: %#v\n", args)
	}
}

func printStmts(title string, stmts []string) {
	printHeader(title)
	for i, s := range stmts {
		fmt.Printf("[%d] %s\n", i+1, s)
	}
}

// --- demos ---

func demoCreateTable(d sqb.Dialect) []string {
	ct := sqb.CreateTable(d).
		Table("public.users").
		IfNotExists().
		Column("id", sqb.BigInt(), sqb.AutoIncrement(), sqb.NotNull()).
		Column("email", sqb.Text(), sqb.NotNull()).
		ColumnStr("name", "TEXT", sqb.NotNull()).
		ColumnStr("status", "TEXT", sqb.DefaultLiteral("'active'")).
		PrimaryKey("id")

	// Indexes: keep one simple and one advanced
	ct.
		UniqueIndex("uidx_users_email", "email").
		AddIndex(func(ix *sqb.CreateIndexBuilder) {
			ix.On("public.users").
				Using("btree").
				Column("name", sqb.Asc()).
				Include("id").
				WhereSQL("status = 'active'")
		})

	// Inline MySQL keys if desired
	if d.Name() == "mysql" {
		ct.InlineMySQLKeys()
	}

	stmts, err := ct.BuildWithIndexes()
	if err != nil {
		return []string{fmt.Sprintf("-- error: %v", err)}
	}
	return stmts
}

func demoInsert(d sqb.Dialect) (string, []any, error) {
	switch d.Name() {
	case "postgres":
		// Your SetExcluded returns map[string]Expr
		set := sqb.SetExcluded(d, "name", "status")
		sql, args, err := sqb.Insert(d).
			Into("public.users").
			Columns("email", "name", "status").
			ValuesRow("a@example.com", "Alice", "active").
			OnConflictDoUpdate(sqb.ConflictColumns("email"), set).
			OnConflictWhere(sqb.NotEq("status", "banned", d)).
			Returning("id").
			Build()
		return sql, args, err

	case "mysql":
		// Your SetFromAlias returns map[string]Expr
		set := sqb.SetFromAlias(d, "v", "name", "status")
		sql, args, err := sqb.Insert(d).
			Into("app.users").
			Columns("email", "name", "status").
			ValuesRow("a@example.com", "Alice", "active").
			MySQLUseAlias("v").
			OnDuplicateKeyUpdate(set).
			Build()
		return sql, args, err

	case "sqlite":
		// Same multi-col helper for SQLite upsert (when enabled in your dialect)
		set := sqb.SetExcluded(d, "name", "status")
		sql, args, err := sqb.Insert(d).
			Into("users").
			Columns("email", "name", "status").
			ValuesRow("a@example.com", "Alice", "active").
			OnConflictDoUpdate(sqb.ConflictColumns("email"), set).
			Build()
		return sql, args, err

	case "sqlserver":
		// MERGE uses explicit Exprs (Raw) for source alias "s"
		set := map[string]sqb.Expr{
			"name":   sqb.RawExpr("s.[name]"),
			"status": sqb.RawExpr("s.[status]"),
		}
		sql, args, err := sqb.Insert(d).
			Into("dbo.Users").
			Columns("email", "name", "status").
			ValuesRow("a@example.com", "Alice", "active").
			MSSQLMergeOn([]string{"email"}, set).
			MSSQLHoldLock().
			OutputInserted("ID").
			Build()
		return sql, args, err
	}
	return "", nil, fmt.Errorf("unsupported dialect: %s", d.Name())
}

func demoSelect(d sqb.Dialect) (string, []any, error) {
	sql, args, err := sqb.Select(d).
		Distinct().
		Columns("id", "email", "name").
		From("public.users").
		Where(sqb.And(
			sqb.NotEq("status", "banned", d),
			sqb.In("id", []any{1, 2, 3}, d),
		)).
		OrderBy("id DESC").
		Limit(50).
		Offset(0).
		Build()
	return sql, args, err
}

func demoUpdate(d sqb.Dialect) (string, []any, error) {
	sql, args, err := sqb.Update(d).
		Table("public.users").
		Set("status", "inactive").
		Where(sqb.Eq("email", "a@example.com", d)).
		Returning("id").
		Build()
	return sql, args, err
}

func demoDelete(d sqb.Dialect) (string, []any, error) {
	sql, args, err := sqb.Delete(d).
		From("public.users").
		Where(sqb.Eq("status", "banned", d)).
		Build()
	return sql, args, err
}

func demoAlterDrop(d sqb.Dialect) ([]string, string) {
	alts, err := sqb.AlterTable(d).
		Table("public.users").
		AddColumn("nickname", sqb.Text()).
		SetDefault("status", "'active'").
		RenameColumn("name", "full_name", sqb.Varchar(100)).
		AddUnique("uq_users_email", "email").
		BuildMany()
	if err != nil {
		alts = []string{fmt.Sprintf("-- alter error: %v", err)}
	}

	drop, _, derr := sqb.DropTable(d).Table("public.temp_users").IfExists().Build()
	if derr != nil {
		drop = fmt.Sprintf("-- drop error: %v", derr)
	}
	return alts, drop
}

func main() {
	// Enable upsert/returning in SQLite demo if your Dialect does so via fields.
	dialects := []sqb.Dialect{
		sqb.Postgres{},
		sqb.MySQL{},
		sqb.SQLite{EnableUpsert: true, EnableReturning: true},
		sqb.SQLServer{},
	}

	for _, d := range dialects {
		fmt.Println()
		fmt.Println("================================================================")
		fmt.Printf("Dialect: %s\n", d.Name())
		fmt.Println("================================================================")

		printStmts("CREATE TABLE + INDEX", demoCreateTable(d))

		if sql, args, err := demoInsert(d); err != nil {
			printSQL(fmt.Sprintf("-- insert error: %v", err), nil)
		} else {
			printHeader("INSERT / UPSERT")
			printSQL(sql, args)
		}

		if sql, args, err := demoSelect(d); err != nil {
			printSQL(fmt.Sprintf("-- select error: %v", err), nil)
		} else {
			printHeader("SELECT")
			printSQL(sql, args)
		}

		if sql, args, err := demoUpdate(d); err != nil {
			printSQL(fmt.Sprintf("-- update error: %v", err), nil)
		} else {
			printHeader("UPDATE")
			printSQL(sql, args)
		}

		if sql, args, err := demoDelete(d); err != nil {
			printSQL(fmt.Sprintf("-- delete error: %v", err), nil)
		} else {
			printHeader("DELETE")
			printSQL(sql, args)
		}

		alts, drop := demoAlterDrop(d)
		printStmts("ALTER TABLE (many)", alts)
		printStmts("DROP TABLE", []string{drop})
	}
}
