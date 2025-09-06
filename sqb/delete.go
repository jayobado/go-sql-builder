package sqb

// ---------- DELETE ----------

type DeleteBuilder struct {
	d                  Dialect
	table              string
	where              []Pred
	outputDeleted      []string // MSSQL
	whereGuardOverride *bool
	metaReason         string
	maxRowsOverride    *int64
}

func Delete(d Dialect) *DeleteBuilder                         { return &DeleteBuilder{d: d} }
func (b *DeleteBuilder) From(table string) *DeleteBuilder     { b.table = table; return b }
func (b *DeleteBuilder) FromStruct(str Struct) *DeleteBuilder { return b.From(str.TableName()) }
func (b *DeleteBuilder) Where(pred Pred) *DeleteBuilder       { b.where = append(b.where, pred); return b }

func (b *DeleteBuilder) OutputDeleted(cols ...string) *DeleteBuilder {
	b.outputDeleted = append(b.outputDeleted, cols...)
	return b
}

func (b *DeleteBuilder) RequireWhere() *DeleteBuilder { t := true; b.whereGuardOverride = &t; return b }
func (b *DeleteBuilder) AllowFullTable() *DeleteBuilder {
	f := false
	b.whereGuardOverride = &f
	return b
}
func (b *DeleteBuilder) AllowFullTableWithMax(max int64) *DeleteBuilder {
	f := false
	b.whereGuardOverride = &f
	b.maxRowsOverride = &max
	return b
}
func (b *DeleteBuilder) Reason(r string) *DeleteBuilder { b.metaReason = r; return b }

func (b *DeleteBuilder) guardMeta() (AuditMeta, int64) {
	max := int64(-1)
	if b.maxRowsOverride != nil {
		max = *b.maxRowsOverride
	}
	return AuditMeta{Op: "DELETE", Table: b.table, Reason: b.metaReason}, max
}

func (b *DeleteBuilder) Build() (string, []any, error) {
	if b.table == "" {
		return "", nil, ErrNoTable
	}
	guard := DefaultSafety.RequireWhereDelete
	if b.whereGuardOverride != nil {
		guard = *b.whereGuardOverride
	}
	if guard && len(b.where) == 0 {
		return "", nil, ErrWhereRequired
	}

	s := &buildState{d: b.d}
	s.write("DELETE FROM ")
	s.write(b.d.QuoteIdent(b.table))

	// MSSQL OUTPUT (immediately after FROM)
	if b.d.Name() == "sqlserver" && len(b.outputDeleted) > 0 {
		s.write(" OUTPUT ")
		for i, c := range b.outputDeleted {
			if i > 0 {
				s.write(", ")
			}
			s.write("DELETED." + b.d.QuoteIdent(c))
		}
	}

	if len(b.where) > 0 {
		s.write(" WHERE ")
		for i, p := range b.where {
			if i > 0 {
				s.write(" AND ")
			}
			s.emitPredicate(p)
		}
	}

	sql, args := s.result()
	return sql, args, nil
}
