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
