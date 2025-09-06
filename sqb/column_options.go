package sqb

type ColOption func(*colDef)

func NotNull() ColOption       { return func(c *colDef) { c.notNull = true } }
func Nullable() ColOption      { return func(c *colDef) { c.notNull = false } }
func PrimaryKey() ColOption    { return func(c *colDef) { c.primary = true } }
func UniqueCol() ColOption     { return func(c *colDef) { c.unique = true } }
func AutoIncrement() ColOption { return func(c *colDef) { c.autoIncrement = true } }

// default literal (no bind args) e.g. "'active'", "NOW()", "CURRENT_TIMESTAMP"
func DefaultLiteral(sql string) ColOption { return func(c *colDef) { c.defaultSQL = sql } }
