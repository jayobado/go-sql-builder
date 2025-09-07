package sqb

import (
	"fmt"
	"strings"
)

type AlterTableBuilder struct {
	d     Dialect
	table string
	ops   []alterOp
}

func AlterTable(d Dialect) *AlterTableBuilder                       { return &AlterTableBuilder{d: d} }
func (b *AlterTableBuilder) Table(name string) *AlterTableBuilder    { b.table = name; return b }
func (b *AlterTableBuilder) AddColumnStr(name, typ string, opts ...ColOption) *AlterTableBuilder {
	c := colDef{name: name, typ: strings.ToUpper(strings.TrimSpace(typ))}
	for _, opt := range opts { opt(&c) }
	b.ops = append(b.ops, addColOp{c: c}); return b
}

func (b *AlterTableBuilder) AddColumn(name string, typ Type, opts ...ColOption) *AlterTableBuilder {
	return b.AddColumnStr(name, typ.SQL(b.d), opts...)
}

func (b *AlterTableBuilder) DropColumn(name string, ifExists bool) *AlterTableBuilder {
	b.ops = append(b.ops, dropColOp{name: name, ifExists: ifExists}); return b
}

func (b *AlterTableBuilder) RenameColumnStr(oldName, newName string) *AlterTableBuilder {
	b.ops = append(b.ops, renameColOp{old: oldName, new: newName}); return b
}

func (b *AlterTableBuilder) RenameColumn(oldName, newName string, newType Type) *AlterTableBuilder {
	switch b.d.Name() {
	case "mysql":
		return b.RenameColumnTyped(oldName, newName, newType.SQL(b.d))
	default:
		return b.RenameColumnStr(oldName, newName)
	}
}

func (b *AlterTableBuilder) RenameColumnTyped(oldName, newName, newType string) *AlterTableBuilder {
	b.ops = append(b.ops, renameColTypedOp{old: oldName, new: newName, typ: newType}); return b
}

func (b *AlterTableBuilder) AlterTypeStr(col, newType string, usingSQL ...string) *AlterTableBuilder {
	var using string; if len(usingSQL) > 0 { using = strings.TrimSpace(usingSQL[0]) }
	b.ops = append(b.ops, alterTypeOp{col: col, typ: strings.ToUpper(strings.TrimSpace(newType)), using: using}); return b
}

func (b *AlterTableBuilder) AlterType(col string, typ Type, usingSQL ...string) *AlterTableBuilder {
	return b.AlterTypeStr(col, typ.SQL(b.d), usingSQL...)
}


func (b *AlterTableBuilder) SetNotNullStr(col string) *AlterTableBuilder {
	b.ops = append(b.ops, setNullOp{col: col, notNull: true}); return b
}
func (b *AlterTableBuilder) SetNullableStr(col string) *AlterTableBuilder {
	b.ops = append(b.ops, setNullOp{col: col, notNull: false}); return b
}
func (b *AlterTableBuilder) SetNullabilityTyped(col, typ string, notNull bool) *AlterTableBuilder {
	b.ops = append(b.ops, setNullTypedOp{col: col, typ: strings.ToUpper(strings.TrimSpace(typ)), notNull: notNull}); return b
}

// SetNullabilityT flips NULL/NOT NULL and supplies the full type when the dialect needs it
// (MySQL/SQL Server). PG/SQLite will fall back to the existing untyped methods.
func (b *AlterTableBuilder) SetNullabilityT(col string, typ Type, notNull bool) *AlterTableBuilder {
	switch b.d.Name() {
	case "mysql", "sqlserver":
		return b.SetNullabilityTyped(col, typ.SQL(b.d), notNull)
	case "postgres":
		if notNull { return b.SetNotNullStr(col) }
		return b.SetNullableStr(col)
	default:
		// SQLite ALTER NULLABILITY isn't supported; keep behavior consistent
		if notNull { return b.SetNotNullStr(col) }
		return b.SetNullableStr(col)
	}
}

func (b *AlterTableBuilder) SetNotNull(col string, typ Type) *AlterTableBuilder {
	switch b.d.Name() {
	case "mysql", "sqlserver":
		return b.SetNullabilityTyped(col, typ.SQL(b.d), true)
	default:
		return b.SetNotNullStr(col)
	}
}

func (b *AlterTableBuilder) SetNullableT(col string, typ Type) *AlterTableBuilder {
	switch b.d.Name() {
	case "mysql", "sqlserver":
		return b.SetNullabilityTyped(col, typ.SQL(b.d), false)
	default:
		return b.SetNullableStr(col)
	}
}

func (b *AlterTableBuilder) SetDefault(col, defaultSQL string) *AlterTableBuilder {
	b.ops = append(b.ops, setDefaultOp{col: col, sql: defaultSQL}); return b
}
func (b *AlterTableBuilder) DropDefault(col string) *AlterTableBuilder {
	b.ops = append(b.ops, dropDefaultOp{col: col}); return b
}
func (b *AlterTableBuilder) AddUnique(name string, cols ...string) *AlterTableBuilder {
	b.ops = append(b.ops, addUniqueOp{name: name, cols: append([]string{}, cols...)}); return b
}
func (b *AlterTableBuilder) DropConstraint(name string) *AlterTableBuilder {
	b.ops = append(b.ops, dropConstraintOp{name: name}); return b
}
func (b *AlterTableBuilder) AddForeignKey(name string, cols []string, refTable string, refCols []string, onDelete, onUpdate string) *AlterTableBuilder {
	b.ops = append(b.ops, addFKOp{name: name, cols: append([]string{}, cols...), refTable: refTable, refCols: append([]string{}, refCols...), onDelete: strings.ToUpper(strings.TrimSpace(onDelete)), onUpdate: strings.ToUpper(strings.TrimSpace(onUpdate))})
	return b
}
func (b *AlterTableBuilder) RenameTable(newName string) *AlterTableBuilder {
	b.ops = append(b.ops, renameTableOp{newName: newName}); return b
}

func (b *AlterTableBuilder) BuildMany() ([]string, error) {
	if b.table == "" { return nil, ErrNoTable }
	var out []string
	for _, op := range b.ops {
		stmts, err := op.render(b.d, b.table)
		if err != nil { return nil, err }
		out = append(out, stmts...)
	}
	return out, nil
}
func (b *AlterTableBuilder) Build() (string, []any, error) {
	stmts, err := b.BuildMany()
	if err != nil { return "", nil, err }
	if len(stmts) == 0 { return "", nil, fmt.Errorf("sqb: no alter operations") }
	if len(stmts) > 1 { return "", nil, fmt.Errorf("sqb: multiple operations; use BuildMany()") }
	return stmts[0], nil, nil
}

// --- internal ops & renderers ---

type alterOp interface{ render(d Dialect, table string) ([]string, error) }

type addColOp struct{ c colDef }
func (op addColOp) render(d Dialect, table string) ([]string, error) {
	qcol := d.QuoteIdent(op.c.name)
	switch d.Name() {
	case "postgres", "sqlite":
		var b strings.Builder
		b.WriteString("ALTER TABLE "); b.WriteString(quoteFQN(d, table)); b.WriteString(" ADD COLUMN "); b.WriteString(qcol)
		b.WriteString(" "); b.WriteString(op.c.typ)
		if op.c.autoIncrement { b.WriteString(" "); b.WriteString(renderAutoInc(d, op.c)) }
		if op.c.notNull { b.WriteString(" NOT NULL") }
		if op.c.defaultSQL != "" { b.WriteString(" DEFAULT "); b.WriteString(op.c.defaultSQL) }
		return []string{b.String() + ";"}, nil
	case "mysql":
		var b strings.Builder
		b.WriteString("ALTER TABLE "); b.WriteString(quoteFQN(d, table)); b.WriteString(" ADD COLUMN "); b.WriteString(qcol)
		b.WriteString(" "); b.WriteString(op.c.typ)
		if op.c.notNull { b.WriteString(" NOT NULL") } else { b.WriteString(" NULL") }
		if op.c.defaultSQL != "" { b.WriteString(" DEFAULT "); b.WriteString(op.c.defaultSQL) }
		if op.c.autoIncrement { b.WriteString(" AUTO_INCREMENT") }
		return []string{b.String() + ";"}, nil
	case "sqlserver":
		var stmts []string
		// add col
		{
			var b strings.Builder
			b.WriteString("ALTER TABLE "); b.WriteString(quoteFQN(d, table)); b.WriteString(" ADD "); b.WriteString(qcol)
			b.WriteString(" "); b.WriteString(op.c.typ)
			if op.c.notNull { b.WriteString(" NOT NULL") } else { b.WriteString(" NULL") }
			stmts = append(stmts, b.String()+";")
		}
		if op.c.defaultSQL != "" {
			dfName := "DF_" + sanitizeNameForConstraint(table) + "_" + sanitizeNameForConstraint(op.c.name)
			stmts = append(stmts, fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s DEFAULT %s FOR %s;", quoteFQN(d, table), d.QuoteIdent(dfName), op.c.defaultSQL, qcol))
		}
		return stmts, nil
	default:
		return nil, ErrAlterOpNotSupported
	}
}

type dropColOp struct{ name string; ifExists bool }
func (op dropColOp) render(d Dialect, table string) ([]string, error) {
	qcol := d.QuoteIdent(op.name)
	switch d.Name() {
	case "postgres":
		if op.ifExists {
			return []string{fmt.Sprintf("ALTER TABLE %s DROP COLUMN IF EXISTS %s;", quoteFQN(d, table), qcol)}, nil
		}
		return []string{fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;", quoteFQN(d, table), qcol)}, nil
	case "mysql", "sqlserver", "sqlite":
		return []string{fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;", quoteFQN(d, table), qcol)}, nil
	default:
		return nil, ErrAlterOpNotSupported
	}
}

type renameColOp struct{ old, new string }
func (op renameColOp) render(d Dialect, table string) ([]string, error) {
	switch d.Name() {
	case "postgres", "sqlite":
		return []string{fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s;", quoteFQN(d, table), d.QuoteIdent(op.old), d.QuoteIdent(op.new))}, nil
	case "sqlserver":
		schema, tbl := splitFQN(table)
		full := fmt.Sprintf("%s.%s.%s", schema, tbl, op.old)
		return []string{fmt.Sprintf("EXEC sp_rename N'%s', N'%s', 'COLUMN';", full, op.new)}, nil
	case "mysql":
		return nil, ErrMySQLTypeRequired
	default:
		return nil, ErrAlterOpNotSupported
	}
}

type renameColTypedOp struct{ old, new, typ string }
func (op renameColTypedOp) render(d Dialect, table string) ([]string, error) {
	switch d.Name() {
	case "mysql":
		return []string{fmt.Sprintf("ALTER TABLE %s CHANGE COLUMN %s %s %s;", quoteFQN(d, table), d.QuoteIdent(op.old), d.QuoteIdent(op.new), op.typ)}, nil
	default:
		return renameColOp{old: op.old, new: op.new}.render(d, table)
	}
}

type alterTypeOp struct{ col, typ, using string }
func (op alterTypeOp) render(d Dialect, table string) ([]string, error) {
	switch d.Name() {
	case "postgres":
		if op.using != "" {
			return []string{fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s USING %s;", quoteFQN(d, table), d.QuoteIdent(op.col), op.typ, op.using)}, nil
		}
		return []string{fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s;", quoteFQN(d, table), d.QuoteIdent(op.col), op.typ)}, nil
	case "mysql":
		return []string{fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s %s;", quoteFQN(d, table), d.QuoteIdent(op.col), op.typ)}, nil
	case "sqlserver":
		return []string{fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s %s;", quoteFQN(d, table), d.QuoteIdent(op.col), op.typ)}, nil
	default:
		return nil, ErrAlterOpNotSupported
	}
}

type setNullOp struct{ col string; notNull bool }
func (op setNullOp) render(d Dialect, table string) ([]string, error) {
	switch d.Name() {
	case "postgres":
		if op.notNull {
			return []string{fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL;", quoteFQN(d, table), d.QuoteIdent(op.col))}, nil
		}
		return []string{fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL;", quoteFQN(d, table), d.QuoteIdent(op.col))}, nil
	default:
		return nil, ErrMySQLTypeRequired
	}
}

type setNullTypedOp struct{ col, typ string; notNull bool }
func (op setNullTypedOp) render(d Dialect, table string) ([]string, error) {
	switch d.Name() {
	case "mysql":
		if op.notNull {
			return []string{fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s %s NOT NULL;", quoteFQN(d, table), d.QuoteIdent(op.col), op.typ)}, nil
		}
		return []string{fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s %s NULL;", quoteFQN(d, table), d.QuoteIdent(op.col), op.typ)}, nil
	case "sqlserver":
		if op.notNull {
			return []string{fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s %s NOT NULL;", quoteFQN(d, table), d.QuoteIdent(op.col), op.typ)}, nil
		}
		return []string{fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s %s NULL;", quoteFQN(d, table), d.QuoteIdent(op.col), op.typ)}, nil
	default:
		return setNullOp{col: op.col, notNull: op.notNull}.render(d, table)
	}
}

type setDefaultOp struct{ col, sql string }
func (op setDefaultOp) render(d Dialect, table string) ([]string, error) {
	switch d.Name() {
	case "postgres":
		return []string{fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET DEFAULT %s;", quoteFQN(d, table), d.QuoteIdent(op.col), op.sql)}, nil
	case "mysql":
		return []string{fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET DEFAULT %s;", quoteFQN(d, table), d.QuoteIdent(op.col), op.sql)}, nil
	case "sqlserver":
		dfName := "DF_" + sanitizeNameForConstraint(table) + "_" + sanitizeNameForConstraint(op.col)
		return []string{fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s DEFAULT %s FOR %s;", quoteFQN(d, table), d.QuoteIdent(dfName), op.sql, d.QuoteIdent(op.col))}, nil
	default:
		return nil, ErrAlterOpNotSupported
	}
}

type dropDefaultOp struct{ col string }
func (op dropDefaultOp) render(d Dialect, table string) ([]string, error) {
	switch d.Name() {
	case "postgres":
		return []string{fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP DEFAULT;", quoteFQN(d, table), d.QuoteIdent(op.col))}, nil
	case "mysql":
		return []string{fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP DEFAULT;", quoteFQN(d, table), d.QuoteIdent(op.col))}, nil
	case "sqlserver":
		schema, tbl := splitFQN(table)
		q := fmt.Sprintf(`
DECLARE @sql nvarchar(max);
SELECT @sql = N'ALTER TABLE %s.%s DROP CONSTRAINT ' + QUOTENAME(dc.name)
FROM sys.default_constraints dc
JOIN sys.columns c ON c.default_object_id = dc.object_id
JOIN sys.tables t ON t.object_id = c.object_id
JOIN sys.schemas s ON s.schema_id = t.schema_id
WHERE s.name = N'%s' AND t.name = N'%s' AND c.name = N'%s';
IF @sql IS NOT NULL EXEC sp_executesql @sql;`, d.QuoteIdent(schema), d.QuoteIdent(tbl), schema, tbl, op.col)
		return []string{q}, nil
	default:
		return nil, ErrAlterOpNotSupported
	}
}

type addUniqueOp struct{ name string; cols []string }
func (op addUniqueOp) render(d Dialect, table string) ([]string, error) {
	if len(op.cols) == 0 { return nil, fmt.Errorf("sqb: unique requires columns") }
	if op.name == "" { return nil, ErrConstraintNameRequired }
	switch d.Name() {
	case "postgres", "sqlserver":
		return []string{fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s UNIQUE (%s);", quoteFQN(d, table), d.QuoteIdent(op.name), joinQuoted(d, op.cols))}, nil
	case "mysql":
		return []string{fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s UNIQUE (%s);", quoteFQN(d, table), d.QuoteIdent(op.name), joinQuoted(d, op.cols))}, nil
	case "sqlite":
		idx := CreateIndex(d).On(table).Name(op.name).Unique()
		for _, c := range op.cols { idx.Column(c) }
		sql, _, err := idx.Build(); if err != nil { return nil, err }
		return []string{sql}, nil
	default:
		return nil, ErrAlterOpNotSupported
	}
}

type dropConstraintOp struct{ name string }
func (op dropConstraintOp) render(d Dialect, table string) ([]string, error) {
	if op.name == "" { return nil, ErrConstraintNameRequired }
	switch d.Name() {
	case "postgres", "sqlserver":
		return []string{fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s;", quoteFQN(d, table), d.QuoteIdent(op.name))}, nil
	case "mysql":
		return []string{fmt.Sprintf("ALTER TABLE %s DROP INDEX %s;", quoteFQN(d, table), d.QuoteIdent(op.name))}, nil
	case "sqlite":
		return []string{fmt.Sprintf("DROP INDEX %s;", d.QuoteIdent(op.name))}, nil
	default:
		return nil, ErrAlterOpNotSupported
	}
}

// --- add FK (missing earlier) ---
type addFKOp struct {
	name     string
	cols     []string
	refTable string
	refCols  []string
	onDelete string
	onUpdate string
}

func (op addFKOp) render(d Dialect, table string) ([]string, error) {
	if op.name == "" {
		return nil, ErrConstraintNameRequired
	}
	if len(op.cols) == 0 || len(op.refCols) == 0 || op.refTable == "" {
		return nil, fmt.Errorf("sqb: foreign key requires cols, refTable, refCols")
	}

	switch d.Name() {
	case "postgres", "mysql", "sqlserver":
		var b strings.Builder
		b.WriteString("ALTER TABLE ")
		b.WriteString(quoteFQN(d, table))
		b.WriteString(" ADD CONSTRAINT ")
		b.WriteString(d.QuoteIdent(op.name))
		b.WriteString(" FOREIGN KEY (")
		b.WriteString(joinQuoted(d, op.cols))
		b.WriteString(") REFERENCES ")
		b.WriteString(quoteFQN(d, op.refTable))
		b.WriteString(" (")
		b.WriteString(joinQuoted(d, op.refCols))
		b.WriteString(")")
		if op.onDelete != "" {
			b.WriteString(" ON DELETE ")
			b.WriteString(op.onDelete)
		}
		if op.onUpdate != "" {
			b.WriteString(" ON UPDATE ")
			b.WriteString(op.onUpdate)
		}
		b.WriteString(";")
		return []string{b.String()}, nil

	case "sqlite":
		// SQLite can't add FKs via ALTER TABLE; requires table rebuild.
		return nil, ErrAlterOpNotSupported
	default:
		return nil, ErrAlterOpNotSupported
	}
}


type renameTableOp struct{ newName string }
func (op renameTableOp) render(d Dialect, table string) ([]string, error) {
	switch d.Name() {
	case "postgres", "sqlite":
		return []string{fmt.Sprintf("ALTER TABLE %s RENAME TO %s;", quoteFQN(d, table), d.QuoteIdent(op.newName))}, nil
	case "mysql":
		return []string{fmt.Sprintf("RENAME TABLE %s TO %s;", quoteFQN(d, table), quoteFQN(d, op.newName))}, nil
	case "sqlserver":
		schema, tbl := splitFQN(table)
		full := fmt.Sprintf("%s.%s", schema, tbl)
		return []string{fmt.Sprintf("EXEC sp_rename N'%s', N'%s', 'OBJECT';", full, op.newName)}, nil
	default:
		return nil, ErrAlterOpNotSupported
	}
}

func sanitizeNameForConstraint(s string) string {
	s = strings.ReplaceAll(s, `"`, "")
	s = strings.ReplaceAll(s, ".", "_")
	return s
}

