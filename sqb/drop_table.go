package sqb

import "fmt"

type DropTableBuilder struct {
	d        Dialect
	table    string
	ifExists bool
	cascade  bool // PG only
}

func DropTable(d Dialect) *DropTableBuilder { return &DropTableBuilder{d: d} }
func (b *DropTableBuilder) Table(name string) *DropTableBuilder { b.table = name; return b }
func (b *DropTableBuilder) IfExists() *DropTableBuilder         { b.ifExists = true; return b }
func (b *DropTableBuilder) Cascade() *DropTableBuilder          { b.cascade = true; return b }

func (b *DropTableBuilder) Build() (string, []any, error) {
	if b.table == "" { return "", nil, ErrNoTable }
	switch b.d.Name() {
	case "postgres":
		sql := "DROP TABLE "
		if b.ifExists { sql += "IF EXISTS " }
		sql += quoteFQN(b.d, b.table)
		if b.cascade { sql += " CASCADE" }
		return sql + ";", nil, nil
	case "mysql", "sqlite":
		sql := "DROP TABLE "
		if b.ifExists { sql += "IF EXISTS " }
		sql += quoteFQN(b.d, b.table)
		return sql + ";", nil, nil
	case "sqlserver":
		if b.ifExists {
			schema, tbl := splitFQN(b.table)
			return fmt.Sprintf("IF OBJECT_ID(N'%s.%s', N'U') IS NOT NULL DROP TABLE %s;", schema, tbl, quoteFQN(b.d, b.table)), nil, nil
		}
		return fmt.Sprintf("DROP TABLE %s;", quoteFQN(b.d, b.table)), nil, nil
	default:
		return "", nil, fmt.Errorf("sqb: unsupported dialect %q", b.d.Name())
	}
}
