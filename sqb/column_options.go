package sqb

type ColOption func(*colDef)

func NotNull() ColOption       { return func(c *colDef) { c.notNull = true } }
func Nullable() ColOption      { return func(c *colDef) { c.notNull = false } }
func PrimaryKey() ColOption    { return func(c *colDef) { c.primary = true } }
func UniqueCol() ColOption     { return func(c *colDef) { c.unique = true } }
func AutoIncrement() ColOption { return func(c *colDef) { c.autoIncrement = true } }

// default literal (no bind args) e.g. "'active'", "NOW()", "CURRENT_TIMESTAMP"
func DefaultLiteral(sql string) ColOption { return func(c *colDef) { c.defaultSQL = sql } }

// in column_options.go (optional sugar)
func UUIDPrimaryKey(d Dialect) ColOption {
	switch d.Name() {
	case "postgres":
		return func(c *colDef) {
			c.typ = "UUID"
			c.primary = true
			c.notNull = true
			c.defaultSQL = "gen_random_uuid()"
		}
	case "mysql":
		return func(c *colDef) {
			c.typ = "CHAR(36)"
			c.primary = true
			c.notNull = true
			c.defaultSQL = "(UUID())"
		}
	case "sqlite":
		return func(c *colDef) {
			c.typ = "TEXT"
			c.primary = true
			c.notNull = true
			c.defaultSQL = "lower(hex(random_blob(16)))"
		}
	default:
		return func(c *colDef) {
			c.typ = "TEXT"
			c.primary = true
			c.notNull = true
		}
	}
}

// timestamp with timezone, default now
func TimestamptzNow(d Dialect) ColOption {
	switch d.Name() {
	case "postgres":
		return func(c *colDef) {
			c.typ = "TIMESTAMPTZ"
			c.defaultSQL = "NOW()"
		}
	case "mysql":
		return func(c *colDef) {
			c.typ = "DATETIME(6)"
			c.defaultSQL = "(CURRENT_TIMESTAMP(6))"
		}
	case "sqlite":
		return func(c *colDef) {
			c.typ = "TEXT"
			c.defaultSQL = "(datetime('now'))"
		}
	case "sqlserver":
		return func(c *colDef) {
			c.typ = "DATETIMEOFFSET"
			c.defaultSQL = "(SYSUTCDATETIME())"
		}
	default:
		return func(c *colDef) {
			c.typ = "TEXT"
		}
	}
}