package sqb

import (
	"fmt"
	"reflect"
	"strings"
)

// StructColumnsValues extracts (columns, values) from a struct using `db` tags,
// skipping any columns listed in exclude (case-insensitive).
//
// Example:
//   cols, vals, _ := StructColumnsValues(user, "id", "created_at")
//   Insert(d).Into("users").Columns(cols...).ValuesRow(vals...)
func StructColumnsValues(v any, exclude ...string) (cols []string, vals []any, err error) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr { rv = rv.Elem() }
	if rv.Kind() != reflect.Struct {
		return nil, nil, fmt.Errorf("sqb: want struct, got %T", v)
	}

	ex := make(map[string]struct{}, len(exclude))
	for _, e := range exclude {
		ex[strings.ToLower(e)] = struct{}{}
	}

	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		col := f.Tag.Get("db")
		if col == "" || col == "-" {
			continue
		}
		if _, skip := ex[strings.ToLower(col)]; skip {
			continue
		}
		cols = append(cols, col)
		vals = append(vals, rv.Field(i).Interface())
	}
	return cols, vals, nil
}

// StructSetMap builds a SET map for UPDATE from a struct using `db` tags,
// skipping any columns listed in exclude.
//
// Example:
//   setMap, _ := StructSetMap(user, "id")
//   up := Update(d).Table("users"); for k,v := range setMap { up.Set(k, v) }
func StructSetMap(v any, exclude ...string) (map[string]any, error) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr { rv = rv.Elem() }
	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("sqb: want struct, got %T", v)
	}

	ex := make(map[string]struct{}, len(exclude))
	for _, e := range exclude {
		ex[strings.ToLower(e)] = struct{}{}
	}

	rt := rv.Type()
	set := make(map[string]any)
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		col := f.Tag.Get("db")
		if col == "" || col == "-" {
			continue
		}
		if _, skip := ex[strings.ToLower(col)]; skip {
			continue
		}
		set[col] = rv.Field(i).Interface()
	}
	return set, nil
}
