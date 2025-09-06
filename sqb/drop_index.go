package sqb

import (
	"fmt"
	"strings"
)

type DropIndexBuilder struct {
	d            Dialect
	name         string
	table        string
	ifExists     bool
	concurrently bool
	cascade      bool
}

func DropIndex(d Dialect) *DropIndexBuilder { return &DropIndexBuilder{d: d} }

func (b *DropIndexBuilder) Name(name string) *DropIndexBuilder { b.name = name; return b }
func (b *DropIndexBuilder) On(table string) *DropIndexBuilder  { b.table = table; return b }
func (b *DropIndexBuilder) IfExists() *DropIndexBuilder        { b.ifExists = true; return b }
func (b *DropIndexBuilder) Concurrently() *DropIndexBuilder    { b.concurrently = true; return b }
func (b *DropIndexBuilder) Cascade() *DropIndexBuilder         { b.cascade = true; return b }

func (b *DropIndexBuilder) Build() (string, []any, error) {
	if b.name == "" { return "", nil, fmt.Errorf("sqb: drop index requires a name") }
	switch b.d.Name() {
	case "postgres":
		var sb strings.Builder
		sb.WriteString("DROP INDEX ")
		if b.concurrently { sb.WriteString("CONCURRENTLY ") }
		if b.ifExists { sb.WriteString("IF EXISTS ") }
		parts := strings.Split(b.name, ".")
		for i, p := range parts {
			if i > 0 { sb.WriteString(".") }
			sb.WriteString(b.d.QuoteIdent(p))
		}
		if b.cascade { sb.WriteString(" CASCADE") }
		sb.WriteString(";")
		return sb.String(), nil, nil
	case "mysql":
		if b.table == "" { return "", nil, ErrIndexTableRequired }
		return fmt.Sprintf("DROP INDEX %s ON %s;", b.d.QuoteIdent(b.name), quoteFQN(b.d, b.table)), nil, nil
	case "sqlite":
		sql := "DROP INDEX "
		if b.ifExists { sql += "IF EXISTS " }
		sql += b.d.QuoteIdent(b.name) + ";"
		return sql, nil, nil
	case "sqlserver":
		if b.table == "" { return "", nil, ErrIndexTableRequired }
		return fmt.Sprintf("DROP INDEX %s ON %s;", b.d.QuoteIdent(b.name), quoteFQN(b.d, b.table)), nil, nil
	}
	return "", nil, fmt.Errorf("sqb: unsupported dialect %q", b.d.Name())
}
