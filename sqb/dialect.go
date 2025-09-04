package sqb

import (
	"fmt"
	"strings"
)

// quote dotted identifiers (schema.table / table.column)
func quotePath(path, left, right string) string {
	if path == "*" {
		return "*"
	}
	if strings.ContainsAny(path, " ()") {
		return path
	}
	parts := strings.Split(path, ".")
	for i, p := range parts {
		parts[i] = left + strings.ReplaceAll(p, right, right+right) + right
	}
	return strings.Join(parts, ".")
}

type Dialect interface {
	Name() string
	Placeholder(n int) string
	QuoteIdent(ident string) string
	HasReturning() bool
	SupportsUpsert() bool
}

// --- Postgres ---
type Postgres struct{}

func (Postgres) Name() string                   { return "postgres" }
func (Postgres) Placeholder(n int) string       { return fmt.Sprintf("$%d", n) }
func (Postgres) HasReturning() bool             { return true }
func (Postgres) SupportsUpsert() bool           { return true }
func (Postgres) QuoteIdent(ident string) string { return quotePath(ident, `"`, `"`) }



// --- MySQL / MariaDB ---
type MySQL struct{}

func (MySQL) Name() string                   { return "mysql" }
func (MySQL) Placeholder(_ int) string       { return "?" }
func (MySQL) HasReturning() bool             { return false }
func (MySQL) SupportsUpsert() bool           { return true }
func (MySQL) QuoteIdent(ident string) string { return quotePath(ident, "`", "`") }



// --- SQLite ---
// Toggle capabilities according to your runtime SQLite version.
type SQLite struct {
	EnableUpsert    bool // 3.24+
	EnableReturning bool // 3.35+
}

func (s SQLite) Name() string                   { return "sqlite" }
func (s SQLite) Placeholder(_ int) string       { return "?" }
func (s SQLite) HasReturning() bool             { return s.EnableReturning }
func (s SQLite) SupportsUpsert() bool           { return s.EnableUpsert }
func (s SQLite) QuoteIdent(ident string) string { return quotePath(ident, `"`, `"`) }


// --- SQL Server ---
type SQLServer struct{}

func (SQLServer) Name() string                   { return "sqlserver" }
func (SQLServer) Placeholder(n int) string       { return fmt.Sprintf("@p%d", n) }
func (SQLServer) HasReturning() bool             { return false } // use OUTPUT
func (SQLServer) SupportsUpsert() bool           { return true }  // via MERGE
func (SQLServer) QuoteIdent(ident string) string { return quotePath(ident, "[", "]") }
