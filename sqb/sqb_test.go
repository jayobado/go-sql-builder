package sqb

import (
	"testing"
)

func TestPostgresDialect(t *testing.T) {
	d := Postgres{}

	if d.Name() != "postgres" {
		t.Errorf("Expected postgres, got %s", d.Name())
	}

	if d.QuoteIdent("table") != `"table"` {
		t.Errorf("Expected quoted table, got %s", d.QuoteIdent("table"))
	}

	if d.QuoteIdent("schema.table") != `"schema"."table"` {
		t.Errorf("Expected quoted schema.table, got %s", d.QuoteIdent("schema.table"))
	}

	if d.Placeholder(1) != "$1" {
		t.Errorf("Expected $1, got %s", d.Placeholder(1))
	}

	if !d.HasReturning() {
		t.Error("Postgres should support RETURNING")
	}

	if !d.SupportsUpsert() {
		t.Error("Postgres should support upsert")
	}
}

func TestMySQLDialect(t *testing.T) {
	d := MySQL{}

	if d.Name() != "mysql" {
		t.Errorf("Expected mysql, got %s", d.Name())
	}

	if d.QuoteIdent("table") != "`table`" {
		t.Errorf("Expected quoted table, got %s", d.QuoteIdent("table"))
	}

	if d.Placeholder(1) != "?" {
		t.Errorf("Expected ?, got %s", d.Placeholder(1))
	}

	if d.HasReturning() {
		t.Error("MySQL should not support RETURNING")
	}

	if !d.SupportsUpsert() {
		t.Error("MySQL should support upsert")
	}
}

func TestSelectBasic(t *testing.T) {
	d := Postgres{}
	sql, args, err := Select(d).
		Columns("id", "email", "name").
		From("users").
		Where(Eq("status", "active", d)).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	expected := `SELECT "id", "email", "name" FROM "users" WHERE ("status" = $1)`
	if sql != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, sql)
	}

	if len(args) != 1 || args[0] != "active" {
		t.Errorf("Expected args [active], got %v", args)
	}
}

func TestSelectWithMultipleWhere(t *testing.T) {
	d := Postgres{}
	sql, args, err := Select(d).
		Columns("id", "email").
		From("users").
		Where(Eq("status", "active", d)).
		Where(NotEq("email", "banned@example.com", d)).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	expected := `SELECT "id", "email" FROM "users" WHERE ("status" = $1) AND ("email" <> $2)`
	if sql != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, sql)
	}

	if len(args) != 2 || args[0] != "active" || args[1] != "banned@example.com" {
		t.Errorf("Expected args [active, banned@example.com], got %v", args)
	}
}

func TestSelectWithAndOr(t *testing.T) {
	d := Postgres{}
	sql, args, err := Select(d).
		Columns("id").
		From("users").
		Where(And(
			Eq("status", "active", d),
			Or(
				Eq("role", "admin", d),
				Eq("role", "moderator", d),
			),
		)).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	expected := `SELECT "id" FROM "users" WHERE (("status" = $1) AND (("role" = $2) OR ("role" = $3)))`
	if sql != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, sql)
	}

	expectedArgs := []any{"active", "admin", "moderator"}
	if len(args) != 3 {
		t.Errorf("Expected 3 args, got %d", len(args))
	}
	for i, expected := range expectedArgs {
		if args[i] != expected {
			t.Errorf("Arg %d: expected %v, got %v", i, expected, args[i])
		}
	}
}

func TestSelectWithIn(t *testing.T) {
	d := Postgres{}
	sql, args, err := Select(d).
		Columns("id").
		From("users").
		Where(In("id", []int{1, 2, 3}, d)).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	expected := `SELECT "id" FROM "users" WHERE ("id" IN ($1,$2,$3))`
	if sql != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, sql)
	}

	if len(args) != 3 || args[0] != 1 || args[1] != 2 || args[2] != 3 {
		t.Errorf("Expected args [1, 2, 3], got %v", args)
	}
}

func TestInsertBasic(t *testing.T) {
	d := Postgres{}
	sql, args, err := Insert(d).
		Into("users").
		Columns("email", "name").
		ValuesRow("alice@example.com", "Alice").
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	expected := `INSERT INTO "users" ("email", "name") VALUES ($1, $2)`
	if sql != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, sql)
	}

	expectedArgs := []any{"alice@example.com", "Alice"}
	if len(args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(args))
	}
	for i, expected := range expectedArgs {
		if args[i] != expected {
			t.Errorf("Arg %d: expected %v, got %v", i, expected, args[i])
		}
	}
}

func TestInsertUpsert(t *testing.T) {
	d := Postgres{}
	sql, args, err := Insert(d).
		Into("users").
		Columns("email", "name").
		ValuesRow("alice@example.com", "Alice").
		OnConflictDoUpdate(
			ConflictColumns("email"),
			SetExcluded(d, "name"),
		).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	expected := `INSERT INTO "users" ("email", "name") VALUES ($1, $2) ON CONFLICT ("email") DO UPDATE SET "name" = EXCLUDED."name"`
	if sql != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, sql)
	}

	expectedArgs := []any{"alice@example.com", "Alice"}
	if len(args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(args))
	}
	for i, expected := range expectedArgs {
		if args[i] != expected {
			t.Errorf("Arg %d: expected %v, got %v", i, expected, args[i])
		}
	}
}

func TestUpdateBasic(t *testing.T) {
	d := Postgres{}
	sql, args, err := Update(d).
		Table("users").
		Set("name", "Alice Updated").
		Where(Eq("id", 1, d)).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	expected := `UPDATE "users" SET "name" = $1 WHERE ("id" = $2)`
	if sql != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, sql)
	}

	expectedArgs := []any{"Alice Updated", 1}
	if len(args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(args))
	}
	for i, expected := range expectedArgs {
		if args[i] != expected {
			t.Errorf("Arg %d: expected %v, got %v", i, expected, args[i])
		}
	}
}

func TestUpdateWithExpr(t *testing.T) {
	d := Postgres{}
	sql, args, err := Update(d).
		Table("users").
		Set("name", "Alice").
		SetExpr("updated_at", RawExpr("NOW()")).
		Where(Eq("id", 1, d)).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	expected := `UPDATE "users" SET "name" = $1, "updated_at" = NOW() WHERE ("id" = $2)`
	if sql != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, sql)
	}

	expectedArgs := []any{"Alice", 1}
	if len(args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(args))
	}
	for i, expected := range expectedArgs {
		if args[i] != expected {
			t.Errorf("Arg %d: expected %v, got %v", i, expected, args[i])
		}
	}
}

func TestDeleteBasic(t *testing.T) {
	d := Postgres{}
	sql, args, err := Delete(d).
		From("users").
		Where(Eq("status", "banned", d)).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	expected := `DELETE FROM "users" WHERE ("status" = $1)`
	if sql != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, sql)
	}

	if len(args) != 1 || args[0] != "banned" {
		t.Errorf("Expected args [banned], got %v", args)
	}
}

func TestSafetyGuards(t *testing.T) {
	d := Postgres{}

	// Save original safety settings
	originalUpdateGuard := DefaultSafety.RequireWhereUpdate
	originalDeleteGuard := DefaultSafety.RequireWhereDelete

	// Enable safety guards
	DefaultSafety.RequireWhereUpdate = true
	DefaultSafety.RequireWhereDelete = true

	defer func() {
		// Restore original settings
		DefaultSafety.RequireWhereUpdate = originalUpdateGuard
		DefaultSafety.RequireWhereDelete = originalDeleteGuard
	}()

	// Test UPDATE without WHERE should fail
	_, _, err := Update(d).
		Table("users").
		Set("status", "disabled").
		Build()

	if err != ErrWhereRequired {
		t.Errorf("Expected ErrWhereRequired, got %v", err)
	}

	// Test DELETE without WHERE should fail
	_, _, err = Delete(d).
		From("users").
		Build()

	if err != ErrWhereRequired {
		t.Errorf("Expected ErrWhereRequired, got %v", err)
	}

	// Test UPDATE with AllowFullTable should succeed
	_, _, err = Update(d).
		Table("users").
		Set("status", "disabled").
		AllowFullTable().
		Build()

	if err != nil {
		t.Errorf("AllowFullTable UPDATE should succeed, got %v", err)
	}

	// Test DELETE with AllowFullTable should succeed
	_, _, err = Delete(d).
		From("users").
		AllowFullTable().
		Build()

	if err != nil {
		t.Errorf("AllowFullTable DELETE should succeed, got %v", err)
	}
}

func TestPredicates(t *testing.T) {
	d := Postgres{}

	// Test NULL handling
	sql, args, err := Select(d).
		Columns("id").
		From("users").
		Where(Eq("deleted_at", nil, d)).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	expected := `SELECT "id" FROM "users" WHERE ("deleted_at" IS NULL)`
	if sql != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args for NULL check, got %v", args)
	}

	// Test NOT NULL handling
	sql, args, err = Select(d).
		Columns("id").
		From("users").
		Where(NotEq("deleted_at", nil, d)).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	expected = `SELECT "id" FROM "users" WHERE ("deleted_at" IS NOT NULL)`
	if sql != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args for NOT NULL check, got %v", args)
	}
}

func TestDialectDifferences(t *testing.T) {
	// Test different dialects produce different SQL
	dialects := []struct {
		name     string
		dialect  Dialect
		expected string
	}{
		{"postgres", Postgres{}, `SELECT "id" FROM "users" WHERE ("status" = $1)`},
		{"mysql", MySQL{}, "SELECT `id` FROM `users` WHERE (`status` = ?)"},
		{"sqlite", SQLite{}, `SELECT "id" FROM "users" WHERE ("status" = ?)`},
		{"sqlserver", SQLServer{}, `SELECT [id] FROM [users] WHERE ([status] = @p1)`},
	}

	for _, test := range dialects {
		t.Run(test.name, func(t *testing.T) {
			sql, args, err := Select(test.dialect).
				Columns("id").
				From("users").
				Where(Eq("status", "active", test.dialect)).
				Build()

			if err != nil {
				t.Fatalf("Build failed: %v", err)
			}

			if sql != test.expected {
				t.Errorf("Expected:\n%s\nGot:\n%s", test.expected, sql)
			}

			if len(args) != 1 || args[0] != "active" {
				t.Errorf("Expected args [active], got %v", args)
			}
		})
	}
}
