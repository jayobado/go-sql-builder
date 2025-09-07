package sqb

import (
	"fmt"
	"strings"
)

// Type is a small interface that can render a SQL type per dialect.
type Type interface {
	SQL(d Dialect) string
}

// -------- Base helpers (portable with smart mappings) --------

// VARCHAR(n)
type tVarchar struct{ N int }

func Varchar(n int) Type { return tVarchar{N: n} }

func (t tVarchar) SQL(d Dialect) string {
	n := t.N
	if n <= 0 {
		n = 255
	}
	switch d.Name() {
	case "sqlite":
		// SQLite ignores length; use TEXT to avoid confusion.
		return "TEXT"
	default:
		return fmt.Sprintf("VARCHAR(%d)", n)
	}
}

// CHAR(n)
type tChar struct{ N int }

func Char(n int) Type { return tChar{N: n} }

func (t tChar) SQL(d Dialect) string {
	n := t.N
	if n <= 0 {
		n = 1
	}
	return fmt.Sprintf("CHAR(%d)", n)
}

// TEXT / CLOB-like
type tText struct{}

func Text() Type { return tText{} }

func (t tText) SQL(d Dialect) string {
	switch d.Name() {
	case "sqlserver":
		// NVARCHAR(MAX) is the practical large text.
		return "NVARCHAR(MAX)"
	default:
		return "TEXT"
	}
}

// BOOLEAN
type tBool struct{}

func Boolean() Type { return tBool{} }

func (t tBool) SQL(d Dialect) string {
	switch d.Name() {
	case "mysql":
		// BOOL alias → TINYINT(1)
		return "TINYINT(1)"
	case "sqlserver":
		return "BIT"
	default:
		return "BOOLEAN"
	}
}

// INTEGER / BIGINT
type tInt struct{}

func Integer() Type { return tInt{} }

func (t tInt) SQL(d Dialect) string {
	switch d.Name() {
	case "sqlserver":
		return "INT"
	default:
		return "INTEGER"
	}
}

type tBigInt struct{}

func BigInt() Type { return tBigInt{} }

func (t tBigInt) SQL(d Dialect) string { return "BIGINT" }

// DECIMAL(p[,s])
type tDecimal struct{ P, S int }

func Decimal(p, s int) Type { return tDecimal{P: p, S: s} }

func (t tDecimal) SQL(d Dialect) string {
	p := t.P
	s := t.S
	if p <= 0 {
		return "DECIMAL"
	}
	if s < 0 {
		return fmt.Sprintf("DECIMAL(%d)", p)
	}
	return fmt.Sprintf("DECIMAL(%d,%d)", p, s)
}


// NUMERIC(p[,s]) — sometimes you prefer numeric
type numeric struct{ P, S int }

func Numeric(p, s int) Type { return numeric{P: p, S: s} }

func (t numeric) SQL(d Dialect) string {
	p := t.P
	s := t.S
	if p <= 0 {
		return "NUMERIC"
	}
	if s < 0 {
		return fmt.Sprintf("NUMERIC(%d)", p)
	}
	return fmt.Sprintf("NUMERIC(%d,%d)", p, s)
}


// DATE
type date struct{}

func Date() Type { return date{} }

func (t date) SQL(d Dialect) string { return "DATE" }



// TIMESTAMP / TIMESTAMPTZ / DATETIME
type timestamp struct{ tz bool }

func Timestamp() Type    { return timestamp{tz: false} } // PG TIMESTAMP, MySQL DATETIME, SQLServer DATETIME2, SQLite TEXT
func Timestamptz() Type  { return timestamp{tz: true} }  // PG TIMESTAMPTZ, SQLServer DATETIMEOFFSET

func (t timestamp) SQL(d Dialect) string {
	switch d.Name() {
	case "postgres":
		if t.tz {
			return "TIMESTAMPTZ"
		}
		return "TIMESTAMP"
	case "mysql":
		// MySQL TIMESTAMP is timezone-naive and narrower; DATETIME is more common for app data.
		return "DATETIME"
	case "sqlserver":
		if t.tz {
			return "DATETIMEOFFSET"
		}
		return "DATETIME2"
	case "sqlite":
		// store as ISO8601 text; SQLite has no native timestamp
		return "TEXT"
	default:
		return "TIMESTAMP"
	}
}

// TIME (with or without TZ hint)
type time struct{ tz bool }

func Time() Type       { return time{tz: false} }
func Timetz() Type     { return time{tz: true} } // PG only in practice

func (t time) SQL(d Dialect) string {
	switch d.Name() {
	case "postgres":
		if t.tz {
			return "TIMETZ"
		}
		return "TIME"
	case "mysql":
		return "TIME"
	case "sqlserver":
		return "TIME"
	case "sqlite":
		return "TEXT"
	default:
		return "TIME"
	}
}


// BINARY/VARBINARY
type varbinary struct{ Length int }

func Varbinary(length int) Type { return varbinary{Length: length} }

func (t varbinary) SQL(d Dialect) string {
	length := t.Length
	if length <= 0 {
		length = 255
	}
	switch d.Name() {
	case "mysql":
		return fmt.Sprintf("VARBINARY(%d)", length)
	case "sqlserver":
		return fmt.Sprintf("VARBINARY(%d)", length)
	default:
		return "BLOB"
	}
}


type binary struct{ Length int }

func Binary(length int) Type { return binary{Length: length} }

func (t binary) SQL(d Dialect) string {
	length := t.Length
	if length <= 0 {
		length = 16
	}
	switch d.Name() {
	case "mysql":
		return fmt.Sprintf("BINARY(%d)", length)
	case "sqlserver":
		return fmt.Sprintf("BINARY(%d)", length)
	default:
		return "BLOB"
	}
}


// JSON / JSONB
type json struct{ Binary bool }

func JSON() Type  { return json{Binary: false} }
func JSONB() Type { return json{Binary: true} }

func (t json) SQL(d Dialect) string {
	switch d.Name() {
	case "postgres":
		if t.Binary {
			return "JSONB"
		}
		return "JSON"
	case "mysql":
		return "JSON"
	case "sqlserver":
		// no native JSON column type; NVARCHAR(MAX) commonly used
		return "NVARCHAR(MAX)"
	case "sqlite":
		// stores as TEXT; declare as JSON for clarity if desired
		return "JSON"
	default:
		return "JSON"
	}
}


// UUID — with optional MySQL Binary(16) mapping
type uuid struct{ Binary16MySQL bool }

func UUID() Type               { return uuid{} }
func UUIDBinary16MySQL() Type  { return uuid{Binary16MySQL: true} }

func (t uuid) SQL(d Dialect) string {
	switch d.Name() {
	case "postgres":
		return "UUID"
	case "mysql":
		if t.Binary16MySQL {
			return "BINARY(16)"
		}
		// human-readable (requires app conversion)
		return "CHAR(36)"
	case "sqlserver":
		return "UNIQUEIDENTIFIER"
	case "sqlite":
		return "TEXT"
	default:
		return "UUID"
	}
}

// ENUM helpers are tricky cross-dialect; provide two targeted variants:

// EnumPG references a pre-created enum type by name (e.g., "status_type").
type enumPG struct{ Name string }

func EnumPG(name string) Type { return enumPG{Name: name} }

func (t enumPG) SQL(d Dialect) string {
	if d.Name() == "postgres" {
		return `"` + strings.ReplaceAll(t.Name, `"`, ``) + `"`
	}
	// Fallback
	return "TEXT"
}

// EnumMySQL inlines values — e.g., EnumMySQL("small","large")
type enumMySQL struct{ Values []string }

func EnumMySQL(values ...string) Type { return enumMySQL{Values: values} }

func (t enumMySQL) SQL(d Dialect) string {
	if d.Name() != "mysql" || len(t.Values) == 0 {
		return "TEXT"
	}
	quoted := make([]string, len(t.Values))
	for i, v := range t.Values {
		quoted[i] = "'" + strings.ReplaceAll(v, "'", "''") + "'"
	}
	return "ENUM(" + strings.Join(quoted, ",") + ")"
}