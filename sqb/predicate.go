package sqb

import (
	"fmt"
	"reflect"
	"strings"
)

type Pred struct {
	sql      string
	arg      []any
	noParens bool
}

func raw(sql string, args ...any) Pred { return Pred{sql: sql, arg: args} }

func wrap(p Pred) string {
	if p.noParens {
		return p.sql
	}
	return "(" + p.sql + ")"
}

// column ops
func Eq(col string, v any, d Dialect) Pred {
	q := d.QuoteIdent(col)
	if v == nil {
		return raw(fmt.Sprintf("%s IS NULL", q))
	}
	return raw(fmt.Sprintf("%s = ?", q), v)
}

func NotEq(col string, v any, d Dialect) Pred {
	q := d.QuoteIdent(col)
	if v == nil {
		return raw(fmt.Sprintf("%s IS NOT NULL", q))
	}
	return raw(fmt.Sprintf("%s <> ?", q), v)
}

func Lt(col string, v any, d Dialect) Pred  { return raw(fmt.Sprintf("%s < ?", d.QuoteIdent(col)), v) }
func Lte(col string, v any, d Dialect) Pred { return raw(fmt.Sprintf("%s <= ?", d.QuoteIdent(col)), v) }

func Gt(col string, v any, d Dialect) Pred  { return raw(fmt.Sprintf("%s > ?", d.QuoteIdent(col)), v) }
func Gte(col string, v any, d Dialect) Pred { return raw(fmt.Sprintf("%s >= ?", d.QuoteIdent(col)), v) }

func In(col string, vals any, d Dialect) Pred {
	rv := reflect.ValueOf(vals)
	if !rv.IsValid() || (rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array) {
		return Eq(col, vals, d)
	}
	if rv.Len() == 0 {
		return raw("1=0")
	}
	ph := make([]string, rv.Len())
	args := make([]any, 0, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		ph[i] = "?"
		args = append(args, rv.Index(i).Interface())
	}
	return raw(fmt.Sprintf("%s IN (%s)", d.QuoteIdent(col), strings.Join(ph, ",")), args...)
}

func NotIn(col string, vals any, d Dialect) Pred {
	rv := reflect.ValueOf(vals)
	if !rv.IsValid() || (rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array) {
		return NotEq(col, vals, d)
	}
	if rv.Len() == 0 {
		return raw("1=1")
	}
	ph := make([]string, rv.Len())
	args := make([]any, 0, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		ph[i] = "?"
		args = append(args, rv.Index(i).Interface())
	}
	return raw(fmt.Sprintf("%s NOT IN (%s)", d.QuoteIdent(col), strings.Join(ph, ",")), args...)
}

func And(preds ...Pred) Pred {
	parts := make([]string, 0, len(preds))
	args := make([]any, 0, len(preds))
	for _, p := range preds {
		parts = append(parts, wrap(p))
		args = append(args, p.arg...)
	}
	return Pred{sql: strings.Join(parts, " AND "), arg: args}
}

func Or(preds ...Pred) Pred {
	parts := make([]string, 0, len(preds))
	args := make([]any, 0, len(preds))
	for _, p := range preds {
		parts = append(parts, wrap(p))
		args = append(args, p.arg...)
	}
	return Pred{sql: strings.Join(parts, " OR "), arg: args}
}

func Not(p Pred) Pred { return Pred{sql: "NOT " + wrap(p), arg: p.arg, noParens: true} }

func IsTrue(col string, d Dialect) Pred  { return raw(fmt.Sprintf("%s IS TRUE", d.QuoteIdent(col))) }
func IsFalse(col string, d Dialect) Pred { return raw(fmt.Sprintf("%s IS FALSE", d.QuoteIdent(col))) }

func IsNull(col string, d Dialect) Pred    { return raw(fmt.Sprintf("%s IS NULL", d.QuoteIdent(col))) }
func IsNotNull(col string, d Dialect) Pred { return raw(fmt.Sprintf("%s IS NOT NULL", d.QuoteIdent(col))) }

func Like(col string, pattern string, d Dialect) Pred {
	return raw(fmt.Sprintf("%s LIKE ?", d.QuoteIdent(col)), pattern)
}

func NotLike(col string, pattern string, d Dialect) Pred {
	return raw(fmt.Sprintf("%s NOT LIKE ?", d.QuoteIdent(col)), pattern)
}

func Between(col string, start, end any, d Dialect) Pred {
	return raw(fmt.Sprintf("%s BETWEEN ? AND ?", d.QuoteIdent(col)), start, end)
}

func NotBetween(col string, start, end any, d Dialect) Pred {
	return raw(fmt.Sprintf("%s NOT BETWEEN ? AND ?", d.QuoteIdent(col)), start, end)
}

func Raw(sql string, args ...any) Pred { return raw(sql, args...) }

func AllRows() Pred { return Pred{sql: "1=1", noParens: true} }