package sqb

import (
	"fmt"
	"sort"
	"strings"
)

// ---------- INSERT ----------

type InsertBuilder struct {
	d         Dialect
	table     string
	cols      []string
	rows      [][]any
	returning []string

	// MSSQL OUTPUT
	outputInserted []string

	// UPSERT
	pgUpsert *pgUpsert
	myUpsert *myUpsert
	msMerge  *msMerge
}

type pgUpsert struct {
	target    *ConflictTarget
	doNothing bool
	setMap    map[string]Expr
	where     []Pred
}

type myUpsert struct {
	doNothing bool
	noopCol   string
	setMap    map[string]Expr
	alias     string
}

type msMerge struct {
	matchCols []string
	updateSet map[string]Expr // nil => no update on match
	holdLock  bool
}

func Insert(d Dialect) *InsertBuilder                            { return &InsertBuilder{d: d} }
func (b *InsertBuilder) Into(table string) *InsertBuilder        { b.table = table; return b }
func (b *InsertBuilder) Columns(cols ...string) *InsertBuilder   { b.cols = append(b.cols, cols...); return b }
func (b *InsertBuilder) ValuesRow(vals ...any) *InsertBuilder    { b.rows = append(b.rows, vals); return b }
func (b *InsertBuilder) ValuesRows(rows [][]any) *InsertBuilder  { b.rows = append(b.rows, rows...); return b }
func (b *InsertBuilder) Returning(cols ...string) *InsertBuilder { b.returning = append(b.returning, cols...); return b }
func (b *InsertBuilder) OutputInserted(cols ...string) *InsertBuilder {
	b.outputInserted = append(b.outputInserted, cols...)
	return b
}

// Postgres / SQLite 3.24+
func (b *InsertBuilder) OnConflictDoNothing(target *ConflictTarget) *InsertBuilder {
	b.pgUpsert = &pgUpsert{doNothing: true, target: target}
	return b
}
func (b *InsertBuilder) OnConflictDoUpdate(target ConflictTarget, set map[string]Expr) *InsertBuilder {
	b.pgUpsert = &pgUpsert{doNothing: false, target: &target, setMap: set}
	return b
}
func (b *InsertBuilder) OnConflictWhere(pred Pred) *InsertBuilder {
	if b.pgUpsert != nil {
		b.pgUpsert.where = append(b.pgUpsert.where, pred)
	}
	return b
}

// MySQL
func (b *InsertBuilder) OnDuplicateKeyUpdate(set map[string]Expr) *InsertBuilder {
	b.myUpsert = &myUpsert{setMap: set}
	return b
}
func (b *InsertBuilder) OnDuplicateKeyDoNothing(noopColumn string) *InsertBuilder {
	b.myUpsert = &myUpsert{doNothing: true, noopCol: noopColumn}
	return b
}
func (b *InsertBuilder) MySQLUseAlias(alias string) *InsertBuilder {
	if b.myUpsert == nil {
		b.myUpsert = &myUpsert{}
	}
	b.myUpsert.alias = alias
	return b
}

// SQL Server MERGE
func (b *InsertBuilder) MSSQLMergeOn(matchCols []string, set map[string]Expr) *InsertBuilder {
	b.msMerge = &msMerge{matchCols: matchCols, updateSet: set}
	return b
}
func (b *InsertBuilder) MSSQLHoldLock() *InsertBuilder {
	if b.msMerge != nil {
		b.msMerge.holdLock = true
	}
	return b
}

func (b *InsertBuilder) Build() (string, []any, error) {
	if b.table == "" {
		return "", nil, ErrNoTable
	}
	if len(b.cols) == 0 {
		return "", nil, ErrNoColumns
	}
	if len(b.rows) == 0 {
		return "", nil, ErrNoRows
	}
	colCount := len(b.cols)
	for i, row := range b.rows {
		if len(row) != colCount {
			return "", nil, fmt.Errorf("sqb: row %d has %d values, expected %d", i, len(row), colCount)
		}
	}

	// SQL Server MERGE path
	if b.d.Name() == "sqlserver" && b.msMerge != nil {
		if len(b.msMerge.matchCols) == 0 {
			return "", nil, ErrNoMatchColumns
		}
		s := &buildState{d: b.d}
		s.write("MERGE INTO ")
		s.write(b.d.QuoteIdent(b.table))
		if b.msMerge.holdLock {
			s.write(" WITH (HOLDLOCK)")
		}
		s.write(" AS ")
		s.write(b.d.QuoteIdent("t"))

		// USING (VALUES ...)
		s.write(" USING (VALUES ")
		for r := 0; r < len(b.rows); r++ {
			if r > 0 { s.write(", ") }
			s.write("(")
			for i := 0; i < colCount; i++ {
				if i > 0 { s.write(", ") }
				s.idx++
				s.write(b.d.Placeholder(s.idx))
			}
			s.write(")")
			s.args = append(s.args, b.rows[r]...)
		}
		s.write(") AS ")
		s.write(b.d.QuoteIdent("s"))
		s.write(" (")

		qcols := make([]string, colCount)
		for i, c := range b.cols { qcols[i] = b.d.QuoteIdent(c) }
		s.write(strings.Join(qcols, ", "))
		s.write(")")

		// ON
		s.write(" ON (")
		for i, k := range b.msMerge.matchCols {
			if i > 0 { s.write(" AND ") }
			qk := b.d.QuoteIdent(k)
			s.write("t." + qk + " = s." + qk)
		}
		s.write(")")

		// WHEN MATCHED
		if len(b.msMerge.updateSet) > 0 {
			s.write(" WHEN MATCHED THEN UPDATE SET ")
			keys := make([]string, 0, len(b.msMerge.updateSet))
			for k := range b.msMerge.updateSet { keys = append(keys, k) }
			sort.Strings(keys)
			for i, k := range keys {
				if i > 0 { s.write(", ") }
				s.write(b.d.QuoteIdent(k) + " = ")
				e := b.msMerge.updateSet[k]
				s.emitSQL(e.sql, e.args)
			}
		}

		// WHEN NOT MATCHED
		s.write(" WHEN NOT MATCHED THEN INSERT (")
		s.write(strings.Join(qcols, ", "))
		s.write(") VALUES (")
		for i, c := range b.cols {
			if i > 0 { s.write(", ") }
			s.write("s." + b.d.QuoteIdent(c))
		}
		s.write(")")

		// OUTPUT
		if len(b.outputInserted) > 0 {
			s.write(" OUTPUT ")
			for i, c := range b.outputInserted {
				if i > 0 { s.write(", ") }
				s.write("INSERTED." + b.d.QuoteIdent(c))
			}
		}
		sql, args := s.result()
		return sql, args, nil
	}

	// Plain INSERT (SQL Server without MERGE still uses OUTPUT)
	if b.d.Name() == "sqlserver" {
		s := &buildState{d: b.d}
		s.write("INSERT INTO ")
		s.write(b.d.QuoteIdent(b.table))
		s.write(" (")

		qcols := make([]string, len(b.cols))
		for i, c := range b.cols { qcols[i] = b.d.QuoteIdent(c) }
		s.write(strings.Join(qcols, ", "))
		s.write(")")

		if len(b.outputInserted) > 0 {
			s.write(" OUTPUT ")
			for i, c := range b.outputInserted {
				if i > 0 { s.write(", ") }
				s.write("INSERTED." + b.d.QuoteIdent(c))
			}
		}

		s.write(" VALUES ")
		for r := 0; r < len(b.rows); r++ {
			if r > 0 { s.write(", ") }
			s.write("(")
			for i := 0; i < colCount; i++ {
				if i > 0 { s.write(", ") }
				s.idx++
				s.write(b.d.Placeholder(s.idx))
			}
			s.write(")")
			s.args = append(s.args, b.rows[r]...)
		}
		sql, args := s.result()
		return sql, args, nil
	}

	// Standard INSERT (PG/MySQL/SQLite)
	s := &buildState{d: b.d}
	s.write("INSERT INTO ")
	s.write(b.d.QuoteIdent(b.table))
	if b.myUpsert != nil && b.myUpsert.alias != "" && b.d.Name() == "mysql" {
		s.write(" AS ")
		s.write(b.d.QuoteIdent(b.myUpsert.alias))
	}
	s.write(" (")

	qcols := make([]string, len(b.cols))
	for i, c := range b.cols { qcols[i] = b.d.QuoteIdent(c) }
	s.write(strings.Join(qcols, ", "))
	s.write(") VALUES ")

	for r := 0; r < len(b.rows); r++ {
		if r > 0 { s.write(", ") }
		s.write("(")
		for i := 0; i < colCount; i++ {
			if i > 0 { s.write(", ") }
			s.idx++
			s.write(b.d.Placeholder(s.idx))
		}
		s.write(")")
		s.args = append(s.args, b.rows[r]...)
	}

	// UPSERT per dialect
	switch b.d.Name() {
	case "postgres":
		if b.pgUpsert != nil {
			if !b.d.SupportsUpsert() { return "", nil, ErrOnConflictNotSupported }
			if b.pgUpsert.doNothing {
				s.write(" ON CONFLICT")
				if b.pgUpsert.target != nil {
					if len(b.pgUpsert.target.Columns) > 0 {
						qs := make([]string, len(b.pgUpsert.target.Columns))
						for i, c := range b.pgUpsert.target.Columns { qs[i] = b.d.QuoteIdent(c) }
						s.write(" (" + strings.Join(qs, ", ") + ")")
					} else if b.pgUpsert.target.Constraint != "" {
						s.write(" ON CONSTRAINT " + b.d.QuoteIdent(b.pgUpsert.target.Constraint))
					}
				}
				s.write(" DO NOTHING")
			} else {
				if b.pgUpsert.target == nil || (len(b.pgUpsert.target.Columns) == 0 && b.pgUpsert.target.Constraint == "") {
					return "", nil, ErrNoConflictTarget
				}
				s.write(" ON CONFLICT")
				if len(b.pgUpsert.target.Columns) > 0 {
					qs := make([]string, len(b.pgUpsert.target.Columns))
					for i, c := range b.pgUpsert.target.Columns { qs[i] = b.d.QuoteIdent(c) }
					s.write(" (" + strings.Join(qs, ", ") + ")")
				} else {
					s.write(" ON CONSTRAINT " + b.d.QuoteIdent(b.pgUpsert.target.Constraint))
				}
				s.write(" DO UPDATE SET ")
				keys := make([]string, 0, len(b.pgUpsert.setMap))
				for k := range b.pgUpsert.setMap { keys = append(keys, k) }
				sort.Strings(keys)
				for i, k := range keys {
					if i > 0 { s.write(", ") }
					s.write(b.d.QuoteIdent(k) + " = ")
					e := b.pgUpsert.setMap[k]
					s.emitSQL(e.sql, e.args)
				}
				if len(b.pgUpsert.where) > 0 {
					s.write(" WHERE ")
					for i, p := range b.pgUpsert.where {
						if i > 0 { s.write(" AND ") }
						s.write(wrap(p))
					}
					for _, p := range b.pgUpsert.where { s.emitPredicate(p) }
				}
			}
		}
	case "sqlite":
		if b.pgUpsert != nil {
			if !b.d.SupportsUpsert() { return "", nil, ErrOnConflictNotSupported }
			if b.pgUpsert.doNothing {
				s.write(" ON CONFLICT")
				if b.pgUpsert.target != nil {
					if len(b.pgUpsert.target.Columns) > 0 {
						qs := make([]string, len(b.pgUpsert.target.Columns))
						for i, c := range b.pgUpsert.target.Columns { qs[i] = b.d.QuoteIdent(c) }
						s.write(" (" + strings.Join(qs, ", ") + ")")
					} else if b.pgUpsert.target.Constraint != "" {
						return "", nil, ErrConstraintNameNotSupported
					}
				}
				s.write(" DO NOTHING")
			} else {
				if b.pgUpsert.target == nil || (len(b.pgUpsert.target.Columns) == 0 && b.pgUpsert.target.Constraint == "") {
					return "", nil, ErrNoConflictTarget
				}
				if b.pgUpsert.target.Constraint != "" {
					return "", nil, ErrConstraintNameNotSupported
				}
				qs := make([]string, len(b.pgUpsert.target.Columns))
				for i, c := range b.pgUpsert.target.Columns { qs[i] = b.d.QuoteIdent(c) }
				s.write(" ON CONFLICT (" + strings.Join(qs, ", ") + ") DO UPDATE SET ")
				keys := make([]string, 0, len(b.pgUpsert.setMap))
				for k := range b.pgUpsert.setMap { keys = append(keys, k) }
				sort.Strings(keys)
				for i, k := range keys {
					if i > 0 { s.write(", ") }
					s.write(b.d.QuoteIdent(k) + " = ")
					e := b.pgUpsert.setMap[k]
					s.emitSQL(e.sql, e.args)
				}
				if len(b.pgUpsert.where) > 0 {
					s.write(" WHERE ")
					for i, p := range b.pgUpsert.where {
						if i > 0 { s.write(" AND ") }
						s.write(wrap(p))
					}
					for _, p := range b.pgUpsert.where { s.emitPredicate(p) }
				}
			}
		}
	case "mysql":
		if b.pgUpsert != nil { return "", nil, ErrOnConflictNotSupported }
		if b.myUpsert != nil {
			s.write(" ON DUPLICATE KEY UPDATE ")
			if b.myUpsert.doNothing {
				if b.myUpsert.noopCol == "" { return "", nil, ErrNoOpColumnRequired }
				qc := b.d.QuoteIdent(b.myUpsert.noopCol)
				s.write(qc + " = " + qc)
			} else {
				keys := make([]string, 0, len(b.myUpsert.setMap))
				for k := range b.myUpsert.setMap { keys = append(keys, k) }
				sort.Strings(keys)
				for i, k := range keys {
					if i > 0 { s.write(", ") }
					s.write(b.d.QuoteIdent(k) + " = ")
					e := b.myUpsert.setMap[k]
					s.emitSQL(e.sql, e.args)
				}
			}
		}
	}

	// RETURNING (PG and optionally SQLite)
	if b.d.HasReturning() && len(b.returning) > 0 {
		s.write(" RETURNING ")
		qr := make([]string, len(b.returning))
		for i, c := range b.returning { qr[i] = b.d.QuoteIdent(c) }
		s.write(strings.Join(qr, ", "))
	}

	sql, args := s.result()
	return sql, args, nil
}

// internal clone helper for batch splitting
func (b *InsertBuilder) valuesRowsCopy(rows [][]any) { b.rows = append(b.rows, rows...) }


// Split into chunks such that placeholders <= maxParams.
// Use e.g. 900 for SQLite, ~60000 for Postgres, etc.
func (b *InsertBuilder) MaxParamsChunk(maxParams int) []*InsertBuilder {
	if maxParams <= 0 || len(b.rows) == 0 || len(b.cols) == 0 {
		return []*InsertBuilder{b}
	}
	perRow := len(b.cols)
	maxRows := maxParams / perRow
	if maxRows <= 0 {
		maxRows = 1
	}
	var chunks []*InsertBuilder
	for i := 0; i < len(b.rows); i += maxRows {
		j := i + maxRows
		if j > len(b.rows) {
			j = len(b.rows)
		}
		nb := Insert(b.d).Into(b.table).Columns(b.cols...)
		if len(b.returning) > 0 {
			nb.Returning(b.returning...)
		}
		if len(b.outputInserted) > 0 {
			nb.OutputInserted(b.outputInserted...)
		}
		nb.valuesRowsCopy(b.rows[i:j])
		// copy UPSERT config
		nb.pgUpsert = b.pgUpsert
		nb.myUpsert = b.myUpsert
		nb.msMerge = b.msMerge
		chunks = append(chunks, nb)
	}
	return chunks
}