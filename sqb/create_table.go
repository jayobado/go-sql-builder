package sqb

import (
	"fmt"
	"strings"
)

type CreateTableBuilder struct {
	d           Dialect
	table       string
	ifNotExists bool
	columns     []colDef
	pk          []string
	uniques     [][]string
	fks         []fkDef

	// indexes
	indexes         []*CreateIndexBuilder
	inlineMySQLKeys bool
}

type colDef struct {
	name          string
	typ           string
	notNull       bool
	defaultSQL    string
	primary       bool
	unique        bool
	autoIncrement bool
}

type fkDef struct {
	cols     []string
	refTable string
	refCols  []string
	onDelete string
	onUpdate string
}

func CreateTable(d Dialect) *CreateTableBuilder { return &CreateTableBuilder{d: d} }

func (b *CreateTableBuilder) Table(name string) *CreateTableBuilder { b.table = name; return b }
func (b *CreateTableBuilder) IfNotExists() *CreateTableBuilder      { b.ifNotExists = true; return b }
func (b *CreateTableBuilder) InlineMySQLKeys() *CreateTableBuilder  { b.inlineMySQLKeys = true; return b }

func (b *CreateTableBuilder) Column(name, sqlType string, opts ...ColOption) *CreateTableBuilder {
	c := colDef{name: name, typ: strings.ToUpper(strings.TrimSpace(sqlType))}
	for _, opt := range opts { opt(&c) }
	b.columns = append(b.columns, c)
	return b
}
func (b *CreateTableBuilder) PrimaryKey(cols ...string) *CreateTableBuilder {
	b.pk = append([]string{}, cols...); return b
}
func (b *CreateTableBuilder) Unique(cols ...string) *CreateTableBuilder {
	if len(cols) > 0 { cp := append([]string{}, cols...); b.uniques = append(b.uniques, cp) }
	return b
}
func (b *CreateTableBuilder) ForeignKey(cols []string, refTable string, refCols []string, onDelete, onUpdate string) *CreateTableBuilder {
	b.fks = append(b.fks, fkDef{
		cols: append([]string{}, cols...),
		refTable: refTable,
		refCols: append([]string{}, refCols...),
		onDelete: strings.ToUpper(strings.TrimSpace(onDelete)),
		onUpdate: strings.ToUpper(strings.TrimSpace(onUpdate)),
	})
	return b
}

// indexes
func (b *CreateTableBuilder) Index(name string, cols ...string) *CreateTableBuilder {
	return b.AddIndex(func(ix *CreateIndexBuilder) {
		ix.Name(name); for _, c := range cols { ix.Column(c) }
	})
}
func (b *CreateTableBuilder) UniqueIndex(name string, cols ...string) *CreateTableBuilder {
	return b.AddIndex(func(ix *CreateIndexBuilder) {
		ix.Name(name).Unique(); for _, c := range cols { ix.Column(c) }
	})
}
func (b *CreateTableBuilder) AddIndex(config func(*CreateIndexBuilder)) *CreateTableBuilder {
	ix := CreateIndex(b.d)
	if b.table != "" { ix.On(b.table) }
	if config != nil { config(ix) }
	b.indexes = append(b.indexes, ix)
	return b
}

func (b *CreateTableBuilder) BuildWithIndexes() ([]string, error) {
	createSQL, _, err := b.Build()
	if err != nil { return nil, err }
	stmts := []string{createSQL}
	for _, ix := range b.indexes {
		if ix.table == "" && b.table != "" { ix.On(b.table) }
		if b.isInlineMySQLEligible(ix) { continue }
		sql, _, err := ix.Build()
		if err != nil { return nil, err }
		stmts = append(stmts, sql)
	}
	return stmts, nil
}

func (b *CreateTableBuilder) Build() (string, []any, error) {
	if b.table == "" { return "", nil, ErrNoTable }
	if len(b.columns) == 0 { return "", nil, fmt.Errorf("sqb: create table requires at least one column") }

	var sb strings.Builder
	switch b.d.Name() {
	case "sqlserver":
		if b.ifNotExists {
			schema, table := splitFQN(b.table)
			sb.WriteString("IF NOT EXISTS (SELECT 1 FROM sys.tables t JOIN sys.schemas s ON s.schema_id=t.schema_id WHERE s.name = '")
			sb.WriteString(schema); sb.WriteString("' AND t.name = '"); sb.WriteString(table); sb.WriteString("')\nBEGIN\n")
		}
		sb.WriteString("CREATE TABLE "); sb.WriteString(quoteFQN(b.d, b.table))
	default:
		sb.WriteString("CREATE TABLE ")
		if b.ifNotExists && (b.d.Name() == "postgres" || b.d.Name() == "sqlite" || b.d.Name() == "mysql") {
			sb.WriteString("IF NOT EXISTS ")
		}
		sb.WriteString(quoteFQN(b.d, b.table))
	}

	sb.WriteString(" (")

	for i, c := range b.columns {
		if i > 0 { sb.WriteString(", ") }
		sb.WriteString(b.d.QuoteIdent(c.name)); sb.WriteString(" "); sb.WriteString(renderColType(b.d, c))
		if c.autoIncrement { sb.WriteString(" "); sb.WriteString(renderAutoInc(b.d, c)) }
		if !(b.d.Name() == "sqlite" && c.autoIncrement) {
			if c.primary { sb.WriteString(" PRIMARY KEY") }
			if c.unique  { sb.WriteString(" UNIQUE") }
			if c.notNull { sb.WriteString(" NOT NULL") }
		}
		if c.defaultSQL != "" { sb.WriteString(" DEFAULT "); sb.WriteString(c.defaultSQL) }
	}

	if len(b.pk) > 0 {
		sb.WriteString(", PRIMARY KEY ("); sb.WriteString(joinQuoted(b.d, b.pk)); sb.WriteString(")")
	}
	for _, grp := range b.uniques {
		sb.WriteString(", UNIQUE ("); sb.WriteString(joinQuoted(b.d, grp)); sb.WriteString(")")
	}
	for _, f := range b.fks {
		sb.WriteString(", FOREIGN KEY ("); sb.WriteString(joinQuoted(b.d, f.cols)); sb.WriteString(") REFERENCES ")
		sb.WriteString(quoteFQN(b.d, f.refTable))
		if len(f.refCols) > 0 { sb.WriteString(" ("); sb.WriteString(joinQuoted(b.d, f.refCols)); sb.WriteString(")") }
		if f.onDelete != "" { sb.WriteString(" ON DELETE "); sb.WriteString(f.onDelete) }
		if f.onUpdate != "" { sb.WriteString(" ON UPDATE "); sb.WriteString(f.onUpdate) }
	}

	// inline MySQL keys
	if b.d.Name() == "mysql" && b.inlineMySQLKeys && len(b.indexes) > 0 {
		for _, ix := range b.indexes {
			if ix.table == "" && b.table != "" { ix.On(b.table) }
			if clause, ok := b.inlineMySQLIndexClause(ix); ok {
				sb.WriteString(", "); sb.WriteString(clause)
			}
		}
	}

	sb.WriteString(")")
	if b.d.Name() == "sqlserver" && b.ifNotExists { sb.WriteString(";\nEND;") } else { sb.WriteString(";") }
	return sb.String(), nil, nil
}

// inline helpers
func (b *CreateTableBuilder) isInlineMySQLEligible(ix *CreateIndexBuilder) bool {
	if b.d.Name() != "mysql" || !b.inlineMySQLKeys || ix == nil { return false }
	if ix.whereSQL != "" || len(ix.include) > 0 { return false }
	for _, p := range ix.parts { if p.isExpr { return false } }
	return true
}
func (b *CreateTableBuilder) inlineMySQLIndexClause(ix *CreateIndexBuilder) (string, bool) {
	if !b.isInlineMySQLEligible(ix) { return "", false }
	if ix.name == "" { ix.name = genIndexName(b.d, ix.table, ix.parts, ix.unique, ix.method) }
	var sb strings.Builder
	if ix.unique { sb.WriteString("UNIQUE ") }
	sb.WriteString("KEY "); sb.WriteString(b.d.QuoteIdent(ix.name))
	if ix.method != "" { sb.WriteString(" USING "); sb.WriteString(strings.ToUpper(ix.method)) }
	sb.WriteString(" (")
	for i, p := range ix.parts {
		if i > 0 { sb.WriteString(", ") }
		sb.WriteString(renderIndexPart(b.d, p))
	}
	sb.WriteString(")")
	return sb.String(), true
}

// helpers
func renderColType(_ Dialect, c colDef) string { return c.typ }

func renderAutoInc(d Dialect, _ colDef) string {
	switch d.Name() {
	case "postgres": return "GENERATED BY DEFAULT AS IDENTITY"
	case "mysql":    return "AUTO_INCREMENT"
	case "sqlserver":return "IDENTITY(1,1)"
	case "sqlite":   return "PRIMARY KEY AUTOINCREMENT"
	default:         return ""
	}
}

func joinQuoted(d Dialect, cols []string) string {
	out := make([]string, len(cols))
	for i, c := range cols { out[i] = d.QuoteIdent(c) }
	return strings.Join(out, ", ")
}
func quoteFQN(d Dialect, fqn string) string {
	parts := strings.Split(fqn, ".")
	for i, p := range parts { parts[i] = d.QuoteIdent(p) }
	return strings.Join(parts, ".")
}
func splitFQN(fqn string) (schema, table string) {
	parts := strings.SplitN(fqn, ".", 2)
	if len(parts) == 1 { return "dbo", parts[0] }
	return parts[0], parts[1]
}
