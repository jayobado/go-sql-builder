// Package sqb provides a type-safe, multi-dialect SQL query builder for Go.
//
// The library supports PostgreSQL, MySQL, SQLite, and SQL Server with
// built-in safety guards, audit capabilities, and advanced features like
// upserts and batch operations.
//
// Example usage:
//
//	d := sqb.Postgres{}
//	sql, args, err := sqb.Select(d).
//		Columns("id", "email", "name").
//		From("users").
//		Where(sqb.Eq("status", "active", d)).
//		Build()
package sqb

import (
	"strconv"
	"strings"
)

// Dialect represents a database dialect with specific SQL syntax and capabilities.
// Each dialect handles identifier quoting, placeholder formatting, and feature
// detection (RETURNING clauses, upsert support, etc.).
type Dialect interface {
	// Name returns the dialect name (postgres, mysql, sqlite, sqlserver).
	Name() string

	// QuoteIdent quotes an identifier (table, column, etc.) according to
	// the dialect's rules. Handles schema.table notation properly.
	QuoteIdent(id string) string

	// Placeholder returns the nth placeholder string for prepared statements.
	// Examples: $1 (Postgres), ? (MySQL/SQLite), @p1 (SQL Server).
	Placeholder(n int) string

	// HasReturning returns true if the dialect supports RETURNING clauses.
	HasReturning() bool

	// SupportsUpsert returns true if the dialect supports upsert operations
	// (ON CONFLICT, ON DUPLICATE KEY UPDATE, MERGE, etc.).
	SupportsUpsert() bool
}

// --- helpers ---

func needsPassthrough(id string) bool {
	// if the caller passed raw SQL with spaces/paren (aliases, functions), don't re-quote
	return strings.ContainsAny(id, " ()\n\t")
}

func quoteSplit(id string, left, right string) string {
	if needsPassthrough(id) {
		return id
	}
	parts := strings.Split(id, ".")
	for i, p := range parts {
		p = strings.ReplaceAll(p, left, "")
		p = strings.ReplaceAll(p, right, "")
		parts[i] = left + p + right
	}
	return strings.Join(parts, ".")
}

// --- Postgres ---

// Postgres implements the PostgreSQL dialect with full feature support.
// Supports RETURNING clauses, ON CONFLICT upserts, and uses $n placeholders
// with double-quoted identifiers.
type Postgres struct{}

func (Postgres) Name() string                { return "postgres" }
func (Postgres) QuoteIdent(id string) string { return quoteSplit(id, `"`, `"`) }
func (Postgres) Placeholder(n int) string    { return "$" + strconv.Itoa(n) }
func (Postgres) HasReturning() bool          { return true }
func (Postgres) SupportsUpsert() bool        { return true }

// --- MySQL ---

// MySQL implements the MySQL dialect with ON DUPLICATE KEY UPDATE support.
// Uses ? placeholders and backtick-quoted identifiers. Does not support
// RETURNING clauses.
type MySQL struct{}

func (MySQL) Name() string                { return "mysql" }
func (MySQL) QuoteIdent(id string) string { return quoteSplit(id, "`", "`") }
func (MySQL) Placeholder(_ int) string    { return "?" }
func (MySQL) HasReturning() bool          { return false }
func (MySQL) SupportsUpsert() bool        { return true }

// --- SQLite ---

// SQLite implements the SQLite dialect with configurable feature support.
// RETURNING and ON CONFLICT support can be enabled based on SQLite version.
// Uses ? placeholders and double-quoted identifiers.
type SQLite struct {
	// EnableUpsert enables ON CONFLICT support (requires SQLite 3.24+).
	EnableUpsert bool
	// EnableReturning enables RETURNING clause support (requires SQLite 3.35+).
	EnableReturning bool
}

func (SQLite) Name() string                { return "sqlite" }
func (SQLite) QuoteIdent(id string) string { return quoteSplit(id, `"`, `"`) }
func (SQLite) Placeholder(_ int) string    { return "?" }
func (s SQLite) HasReturning() bool        { return s.EnableReturning }
func (s SQLite) SupportsUpsert() bool      { return s.EnableUpsert }

// --- SQL Server ---

// SQLServer implements the SQL Server dialect with MERGE and OUTPUT support.
// Uses @pn placeholders and bracket-quoted identifiers. Supports advanced
// features like MERGE statements and OUTPUT clauses.
type SQLServer struct{}

func (SQLServer) Name() string                { return "sqlserver" }
func (SQLServer) QuoteIdent(id string) string { return quoteSplit(id, "[", "]") }
func (SQLServer) Placeholder(n int) string    { return "@p" + strconv.Itoa(n) }
func (SQLServer) HasReturning() bool          { return false }
func (SQLServer) SupportsUpsert() bool        { return false }
