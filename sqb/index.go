package sqb

import (
	"errors"
	"fmt"
	"hash/crc32"
	"strings"
)

type CreateIndexBuilder struct {
	d            Dialect
	name         string
	table        string
	unique       bool
	method       string
	concurrently bool
	ifNotExists  bool
	clustered    *bool
	parts        []IndexPart
	include      []string
	whereSQL     string
	withRaw      string
}

type IndexPart struct {
	expr    string
	isExpr  bool
	desc    bool
	nulls   string
	length  int
	collate string
}

func CreateIndex(d Dialect) *CreateIndexBuilder { return &CreateIndexBuilder{d: d} }

func (b *CreateIndexBuilder) Name(name string) *CreateIndexBuilder { b.name = name; return b }
func (b *CreateIndexBuilder) On(table string) *CreateIndexBuilder  { b.table = table; return b }
func (b *CreateIndexBuilder) Unique() *CreateIndexBuilder          { b.unique = true; return b }
func (b *CreateIndexBuilder) Using(method string) *CreateIndexBuilder {
	b.method = strings.ToLower(strings.TrimSpace(method)); return b
}
func (b *CreateIndexBuilder) Concurrently() *CreateIndexBuilder { b.concurrently = true; return b }
func (b *CreateIndexBuilder) IfNotExists() *CreateIndexBuilder  { b.ifNotExists = true; return b }
func (b *CreateIndexBuilder) Clustered() *CreateIndexBuilder    { t := true; b.clustered = &t; return b }
func (b *CreateIndexBuilder) NonClustered() *CreateIndexBuilder { f := false; b.clustered = &f; return b }

func (b *CreateIndexBuilder) Column(col string, opts ...IndexColOption) *CreateIndexBuilder {
	p := IndexPart{expr: col, isExpr: false}
	for _, o := range opts { o(&p) }
	b.parts = append(b.parts, p); return b
}
func (b *CreateIndexBuilder) Expr(expr string, opts ...IndexColOption) *CreateIndexBuilder {
	p := IndexPart{expr: expr, isExpr: true}
	for _, o := range opts { o(&p) }
	b.parts = append(b.parts, p); return b
}
func (b *CreateIndexBuilder) Include(cols ...string) *CreateIndexBuilder {
	b.include = append(b.include, cols...); return b
}
func (b *CreateIndexBuilder) WhereSQL(sql string) *CreateIndexBuilder {
	b.whereSQL = strings.TrimSpace(sql); return b
}
func (b *CreateIndexBuilder) WithRaw(raw string) *CreateIndexBuilder {
	b.withRaw = strings.TrimSpace(raw); return b
}

func (b *CreateIndexBuilder) Build() (string, []any, error) {
	if b.table == "" { return "", nil, ErrNoTable }
	if len(b.parts) == 0 { return "", nil, errors.New("sqb: index requires at least one key part") }
	if b.name == "" { b.name = genIndexName(b.d, b.table, b.parts, b.unique, b.method) }

	var sb strings.Builder
	switch b.d.Name() {
	case "postgres":
		sb.WriteString("CREATE ")
		if b.unique { sb.WriteString("UNIQUE ") }
		sb.WriteString("INDEX ")
		if b.concurrently { sb.WriteString("CONCURRENTLY ") }
		if b.ifNotExists { sb.WriteString("IF NOT EXISTS ") }
		sb.WriteString(b.d.QuoteIdent(b.name))
		sb.WriteString(" ON "); sb.WriteString(quoteFQN(b.d, b.table))
		if b.method != "" { sb.WriteString(" USING "); sb.WriteString(strings.ToUpper(b.method)) }
		sb.WriteString(" (")
		for i, p := range b.parts {
			if i > 0 { sb.WriteString(", ") }
			sb.WriteString(renderIndexPart(b.d, p))
		}
		sb.WriteString(")")
		if len(b.include) > 0 {
			sb.WriteString(" INCLUDE ("); sb.WriteString(joinQuoted(b.d, b.include)); sb.WriteString(")")
		}
		if b.withRaw != "" { sb.WriteString(" WITH ("); sb.WriteString(b.withRaw); sb.WriteString(")") }
		if b.whereSQL != "" { sb.WriteString(" WHERE "); sb.WriteString(b.whereSQL) }
		sb.WriteString(";")
		return sb.String(), nil, nil

	case "mysql":
		if b.whereSQL != "" { return "", nil, ErrIndexWhereNotSupported }
		if len(b.include) > 0 { return "", nil, ErrIndexIncludeNotSupported }
		sb.WriteString("CREATE ")
		if b.unique { sb.WriteString("UNIQUE ") }
		sb.WriteString("INDEX "); sb.WriteString(b.d.QuoteIdent(b.name))
		if b.method != "" { sb.WriteString(" USING "); sb.WriteString(strings.ToUpper(b.method)) }
		sb.WriteString(" ON "); sb.WriteString(quoteFQN(b.d, b.table)); sb.WriteString(" (")
		for i, p := range b.parts {
			if i > 0 { sb.WriteString(", ") }
			sb.WriteString(renderIndexPart(b.d, p))
		}
		sb.WriteString(");")
		return sb.String(), nil, nil

	case "sqlite":
		sb.WriteString("CREATE ")
		if b.unique { sb.WriteString("UNIQUE ") }
		sb.WriteString("INDEX ")
		if b.ifNotExists { sb.WriteString("IF NOT EXISTS ") }
		sb.WriteString(b.d.QuoteIdent(b.name))
		sb.WriteString(" ON "); sb.WriteString(quoteFQN(b.d, b.table)); sb.WriteString(" (")
		for i, p := range b.parts {
			if i > 0 { sb.WriteString(", ") }
			sb.WriteString(renderIndexPart(b.d, p))
		}
		sb.WriteString(")")
		if b.whereSQL != "" { sb.WriteString(" WHERE "); sb.WriteString(b.whereSQL) }
		sb.WriteString(";")
		return sb.String(), nil, nil

	case "sqlserver":
		sb.WriteString("CREATE ")
		if b.unique { sb.WriteString("UNIQUE ") }
		if b.clustered != nil {
			if *b.clustered { sb.WriteString("CLUSTERED ") } else { sb.WriteString("NONCLUSTERED ") }
		}
		sb.WriteString("INDEX "); sb.WriteString(b.d.QuoteIdent(b.name))
		sb.WriteString(" ON "); sb.WriteString(quoteFQN(b.d, b.table)); sb.WriteString(" (")
		for i, p := range b.parts {
			if i > 0 { sb.WriteString(", ") }
			sb.WriteString(renderIndexPart(b.d, p))
		}
		sb.WriteString(")")
		if len(b.include) > 0 { sb.WriteString(" INCLUDE ("); sb.WriteString(joinQuoted(b.d, b.include)); sb.WriteString(")") }
		if b.whereSQL != "" { sb.WriteString(" WHERE "); sb.WriteString(b.whereSQL) }
		if b.withRaw != "" { sb.WriteString(" WITH ("); sb.WriteString(b.withRaw); sb.WriteString(")") }
		sb.WriteString(";")
		return sb.String(), nil, nil
	}
	return "", nil, fmt.Errorf("sqb: unsupported dialect %q", b.d.Name())
}

func renderIndexPart(d Dialect, p IndexPart) string {
	var b strings.Builder
	if p.isExpr { b.WriteString(p.expr) } else { b.WriteString(d.QuoteIdent(p.expr)) }
	if p.length > 0 && d.Name() == "mysql" { b.WriteString(fmt.Sprintf("(%d)", p.length)) }
	if p.collate != "" && (d.Name() == "postgres" || d.Name() == "mysql") {
		b.WriteString(" COLLATE "); b.WriteString(p.collate)
	}
	if p.desc { b.WriteString(" DESC") } else if d.Name() == "sqlserver" { b.WriteString(" ASC") }
	if d.Name() == "postgres" && (p.nulls == "FIRST" || p.nulls == "LAST") {
		b.WriteString(" NULLS "); b.WriteString(p.nulls)
	}
	return b.String()
}

func genIndexName(d Dialect, table string, parts []IndexPart, unique bool, method string) string {
	base := "idx"; if unique { base = "uidx" }
	tbl := table
	if i := strings.LastIndex(table, "."); i >= 0 && i+1 < len(table) {
		tbl = table[i+1:]
	}
	tbl = strings.ToLower(strings.ReplaceAll(tbl, `"`, ""))
	colBits := make([]string, 0, len(parts))
	for _, p := range parts {
		x := p.expr
		if !p.isExpr { x = strings.ToLower(strings.ReplaceAll(x, `"`, "")) } else { x = "expr" }
		if p.length > 0 { x += fmt.Sprintf("_%d", p.length) }
		if p.desc { x += "_desc" }
		colBits = append(colBits, x)
	}
	name := base + "_" + tbl + "_" + strings.Join(colBits, "_")
	if method != "" && d.Name() == "postgres" { name += "_" + strings.ToLower(method) }

	max := 64
	switch d.Name() {
	case "postgres": max = 63
	case "mysql": max = 64
	case "sqlserver": max = 128
	case "sqlite": max = 64
	}
	if len(name) <= max { return name }
	sum := crc32.ChecksumIEEE([]byte(name))
	hash := fmt.Sprintf("%08x", sum)
	keep := max - (1 + len(hash))
	if keep < 8 { keep = 8 }
	return name[:keep] + "_" + hash
}
