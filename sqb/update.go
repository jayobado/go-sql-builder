package sqb

import (
	"sort"
	"strings"
)

// ---------- UPDATE ----------

type UpdateBuilder struct {
	d                  Dialect
	table              string
	setMap             map[string]any
	where              []Pred
	returning          []string
	outputInserted     []string // MSSQL
	outputDeleted      []string // MSSQL
	whereGuardOverride *bool
	metaReason         string
	maxRowsOverride    *int64
}

func Update(d Dialect) *UpdateBuilder                           { return &UpdateBuilder{d: d, setMap: map[string]any{}} }
func (b *UpdateBuilder) Table(table string) *UpdateBuilder      { b.table = table; return b }
func (b *UpdateBuilder) Set(col string, val any) *UpdateBuilder { b.setMap[col] = val; return b }
func (b *UpdateBuilder) Where(pred Pred) *UpdateBuilder         { b.where = append(b.where, pred); return b }

func (b *UpdateBuilder) Returning(cols ...string) *UpdateBuilder {
	b.returning = append(b.returning, cols...); return b
}
func (b *UpdateBuilder) OutputInserted(cols ...string) *UpdateBuilder {
	b.outputInserted = append(b.outputInserted, cols...); return b
}
func (b *UpdateBuilder) OutputDeleted(cols ...string) *UpdateBuilder {
	b.outputDeleted = append(b.outputDeleted, cols...); return b
}

func (b *UpdateBuilder) RequireWhere() *UpdateBuilder         { t := true;  b.whereGuardOverride = &t; return b }
func (b *UpdateBuilder) AllowFullTable() *UpdateBuilder       { f := false; b.whereGuardOverride = &f; return b }
func (b *UpdateBuilder) AllowFullTableWithMax(max int64) *UpdateBuilder {
	f := false; b.whereGuardOverride = &f; b.maxRowsOverride = &max; return b
}
func (b *UpdateBuilder) Reason(r string) *UpdateBuilder { b.metaReason = r; return b }

func (b *UpdateBuilder) guardMeta() (AuditMeta, int64) {
	max := int64(-1)
	if b.maxRowsOverride != nil { max = *b.maxRowsOverride }
	return AuditMeta{Op: "UPDATE", Table: b.table, Reason: b.metaReason}, max
}

func (b *UpdateBuilder) Build() (string, []any, error) {
	if b.table == "" {
		return "", nil, ErrNoTable
	}
	if len(b.setMap) == 0 {
		return "", nil, ErrNoSetClauses
	}

	guard := DefaultSafety.RequireWhereUpdate
	if b.whereGuardOverride != nil { guard = *b.whereGuardOverride }
	if guard && len(b.where) == 0 {
		return "", nil, ErrWhereRequired
	}

	s := &buildState{d: b.d}
	s.write("UPDATE ")
	s.write(b.d.QuoteIdent(b.table))
	s.write(" SET ")

	keys := make([]string, 0, len(b.setMap))
	for k := range b.setMap { keys = append(keys, k) }
	sort.Strings(keys)

	for i, k := range keys {
		if i > 0 { s.write(", ") }
		s.write(b.d.QuoteIdent(k))
		s.write(" = ")
		s.idx++
		s.write(b.d.Placeholder(s.idx))
		s.args = append(s.args, b.setMap[k])
	}

	// MSSQL OUTPUT (after SET, before WHERE)
	if b.d.Name() == "sqlserver" && (len(b.outputInserted) > 0 || len(b.outputDeleted) > 0) {
		s.write(" OUTPUT ")
		first := true
		emit := func(prefix string, cols []string) {
			for _, c := range cols {
				if !first { s.write(", ") }
				first = false
				s.write(prefix + "." + b.d.QuoteIdent(c))
			}
		}
		emit("INSERTED", b.outputInserted)
		emit("DELETED", b.outputDeleted)
	}

	if len(b.where) > 0 {
		s.write(" WHERE ")
		for i, p := range b.where {
			if i > 0 { s.write(" AND ") }
			s.write(wrap(p))
		}
		for _, p := range b.where { s.emitPredicate(p) }
	}

	if b.d.HasReturning() && len(b.returning) > 0 {
		s.write(" RETURNING ")
		qr := make([]string, len(b.returning))
		for i, c := range b.returning { qr[i] = b.d.QuoteIdent(c) }
		s.write(strings.Join(qr, ", "))
	}

	sql, args := s.result()
	return sql, args, nil
}
