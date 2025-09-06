package sqb

import (
	"strings"
	"strconv"
)

type Dialect interface {
	Name() string
	QuoteIdent(id string) string
	Placeholder(n int) string
	HasReturning() bool
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

type Postgres struct{}

func (Postgres) Name() string                    { return "postgres" }
func (Postgres) QuoteIdent(id string) string     { return quoteSplit(id, `"`, `"`) }
func (Postgres) Placeholder(n int) string        { return "$" + strconv.Itoa(n) }
func (Postgres) HasReturning() bool              { return true }
func (Postgres) SupportsUpsert() bool            { return true }

// --- MySQL ---

type MySQL struct{}

func (MySQL) Name() string                 { return "mysql" }
func (MySQL) QuoteIdent(id string) string  { return quoteSplit(id, "`", "`") }
func (MySQL) Placeholder(_ int) string     { return "?" }
func (MySQL) HasReturning() bool           { return false }
func (MySQL) SupportsUpsert() bool         { return true }

// --- SQLite ---

type SQLite struct {
	EnableUpsert    bool
	EnableReturning bool
}

func (SQLite) Name() string                  { return "sqlite" }
func (SQLite) QuoteIdent(id string) string   { return quoteSplit(id, `"`, `"`) }
func (SQLite) Placeholder(_ int) string      { return "?" }
func (s SQLite) HasReturning() bool          { return s.EnableReturning }
func (s SQLite) SupportsUpsert() bool        { return s.EnableUpsert }

// --- SQL Server ---

type SQLServer struct{}

func (SQLServer) Name() string                 { return "sqlserver" }
func (SQLServer) QuoteIdent(id string) string  { return quoteSplit(id, "[", "]") }
func (SQLServer) Placeholder(n int) string     { return "@p" + strconv.Itoa(n) }
func (SQLServer) HasReturning() bool           { return false }
func (SQLServer) SupportsUpsert() bool         { return false }
