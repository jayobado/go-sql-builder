package sqb

import (
	"strings"
	"strconv"
)

type Struct interface {
	TableName() string
}

type SelectBuilder struct {
	d        Dialect
	distinct bool
	cols     []string
	rawCols  []string
	table    string
	joins    []string
	where    []Pred
	groupBy  []string
	having   []Pred
	orderBy  []string
	limit    *int
	offset   *int

	requireFrom  *bool
	requireLimit *bool
}

func Select(d Dialect) *SelectBuilder { return &SelectBuilder{d: d} }

func (b *SelectBuilder) Distinct() *SelectBuilder                 { b.distinct = true; return b }
func (b *SelectBuilder) Columns(cols ...string) *SelectBuilder    { b.cols = append(b.cols, cols...); return b }
func (b *SelectBuilder) ColumnExpr(expr string) *SelectBuilder    { b.rawCols = append(b.rawCols, expr); return b }
func (b *SelectBuilder) From(table string) *SelectBuilder         { b.table = table; return b }
func (b *SelectBuilder) FromStruct(str Struct) *SelectBuilder     { return b.From(str.TableName()) }
func (b *SelectBuilder) Join(joinSQL string) *SelectBuilder       { b.joins = append(b.joins, joinSQL); return b }
func (b *SelectBuilder) Where(pred Pred) *SelectBuilder           { b.where = append(b.where, pred); return b }
func (b *SelectBuilder) WhereRaw(sql string, args ...any) *SelectBuilder {
	b.where = append(b.where, raw(sql, args...)); return b
}
func (b *SelectBuilder) GroupBy(cols ...string) *SelectBuilder    { b.groupBy = append(b.groupBy, cols...); return b }
func (b *SelectBuilder) Having(pred Pred) *SelectBuilder          { b.having = append(b.having, pred); return b }
func (b *SelectBuilder) HavingRaw(sql string, args ...any) *SelectBuilder {
	b.having = append(b.having, raw(sql, args...)); return b
}
func (b *SelectBuilder) OrderBy(exprs ...string) *SelectBuilder   { b.orderBy = append(b.orderBy, exprs...); return b }
func (b *SelectBuilder) Limit(n int) *SelectBuilder               { b.limit = &n; return b }
func (b *SelectBuilder) Offset(n int) *SelectBuilder              { b.offset = &n; return b }
func (b *SelectBuilder) RequireFrom() *SelectBuilder              { t := true; b.requireFrom = &t; return b }
func (b *SelectBuilder) RequireLimit() *SelectBuilder             { t := true; b.requireLimit = &t; return b }

func (b *SelectBuilder) Build() (string, []any, error) {
	if len(b.joins) > 0 && b.table == "" {
		return "", nil, ErrJoinWithoutFrom
	}
	if b.limit != nil && *b.limit < 0 {
		return "", nil, ErrNegativeLimit
	}
	if b.offset != nil && *b.offset < 0 {
		return "", nil, ErrNegativeOffset
	}
	if (b.requireFrom != nil && *b.requireFrom) || DefaultSafety.RequireFromSelect {
		if b.table == "" {
			return "", nil, ErrFromRequired
		}
	}
	if (b.requireLimit != nil && *b.requireLimit) || DefaultSafety.RequireLimitSelect {
		if b.limit == nil {
			return "", nil, ErrLimitRequired
		}
	}

	s := &buildState{d: b.d}
	s.write("SELECT ")
	if b.distinct {
		s.write("DISTINCT ")
	}
	
	if len(b.cols) == 0 && len(b.rawCols) == 0 {
		s.write("*")
	} else {
		parts := make([]string, 0, len(b.cols)+len(b.rawCols))
		for _, c := range b.cols {
			parts = append(parts, b.d.QuoteIdent(c))
		}
		parts = append(parts, b.rawCols...)
		s.write(strings.Join(parts, ", "))
	}

	if b.table != "" {
		s.write(" FROM ")
		s.write(b.d.QuoteIdent(b.table))
	}
	for _, j := range b.joins {
		s.write(" ")
		s.write(j)
	}
	if len(b.where) > 0 {
		s.write(" WHERE ")
		for i, p := range b.where {
			if i > 0 {
				s.write(" AND ")
			}
			s.write(wrap(p))
		}
		for _, p := range b.where {
			s.emitPredicate(p)
		}
	}
	if len(b.groupBy) > 0 {
		s.write(" GROUP BY ")
		q := make([]string, len(b.groupBy))
		for i, c := range b.groupBy {
			q[i] = b.d.QuoteIdent(c)
		}
		s.write(strings.Join(q, ", "))
	}
	if len(b.having) > 0 {
		s.write(" HAVING ")
		for i, p := range b.having {
			if i > 0 {
				s.write(" AND ")
			}
			s.write(wrap(p))
		}
		for _, p := range b.having {
			s.emitPredicate(p)
		}
	}
	if len(b.orderBy) > 0 {
		s.write(" ORDER BY ")
		s.write(strings.Join(b.orderBy, ", "))
	}
	if b.limit != nil {
		s.write(" LIMIT ")
		s.write(strconv.Itoa(*b.limit))
	}
	if b.offset != nil {
		s.write(" OFFSET ")
		s.write(strconv.Itoa(*b.offset))
	}
	sql, args := s.result()
	return sql, args, nil
}

