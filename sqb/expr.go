package sqb

type Expr struct {
	sql  string
	args []any
}

func Val(v any) Expr                   { return Expr{sql: "?", args: []any{v}} }
func RawExpr(sql string, args ...any) Expr { return Expr{sql: sql, args: args} }

// EXCLUDED / excluded (PG/SQLite)
func Excluded(col string, d Dialect) Expr {
	switch d.Name() {
	case "postgres":
		return Expr{sql: "EXCLUDED." + d.QuoteIdent(col)}
	case "sqlite":
		return Expr{sql: "excluded." + d.QuoteIdent(col)}
	default:
		return Expr{sql: "EXCLUDED." + d.QuoteIdent(col)}
	}
}

// Legacy MySQL helper (prefer AliasCol with INSERT alias).
func Values(col string, d Dialect) Expr {
	return Expr{sql: "VALUES(" + d.QuoteIdent(col) + ")"}
}

func AliasCol(alias, col string, d Dialect) Expr {
	return Expr{sql: d.QuoteIdent(alias + "." + col)}
}

// Sugar for deterministic SET maps:
func SetExcluded(d Dialect, cols ...string) map[string]Expr {
	m := make(map[string]Expr, len(cols))
	for _, c := range cols {
		m[c] = Excluded(c, d)
	}
	return m
}
func SetFromAlias(d Dialect, alias string, cols ...string) map[string]Expr {
	m := make(map[string]Expr, len(cols))
	for _, c := range cols {
		m[c] = AliasCol(alias, c, d)
	}
	return m
}
func MergeSets(sets ...map[string]Expr) map[string]Expr {
	out := map[string]Expr{}
	for _, s := range sets {
		for k, v := range s {
			out[k] = v
		}
	}
	return out
}

// Default() lets you put the SQL keyword DEFAULT as a value in INSERT rows.
type _defaultSentinel struct{}
var _defaultValue = _defaultSentinel{}

func Default() any { return _defaultValue }

// UUIDv4 returns a dialect-appropriate UUID expression as an Expr.
func UUIDv4(d Dialect) Expr {
	switch d.Name() {
	case "postgres":
		return RawExpr("gen_random_uuid()") // requires pgcrypto extension
	case "mysql":
		return RawExpr("UUID()")            // VARCHAR(36); for BINARY(16) you'd use UUID_TO_BIN(...)
	case "sqlserver":
		return RawExpr("NEWID()")
	case "sqlite":
		// a common SQL-only UUID v4 expression approximation
		return RawExpr(`lower(hex(randomblob(4)) || '-' || hex(randomblob(2)) || '-4' ||
		            substr(hex(randomblob(2)),2) || '-' ||
		            substr('89ab',abs(random()) % 4 + 1,1) ||
		            substr(hex(randomblob(2)),2) || '-' || hex(randomblob(6)))`)
	default:
		return RawExpr("UUID()")
	}
}