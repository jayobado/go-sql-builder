package sqb

type TruncateBuilder struct {
	d               Dialect
	table           string
	restartIdentity bool // PG
	cascade         bool // PG
	metaReason      string
}

func Truncate(d Dialect) *TruncateBuilder                   { return &TruncateBuilder{d: d} }
func (b *TruncateBuilder) Table(t string) *TruncateBuilder  { b.table = t; return b }
func (b *TruncateBuilder) RestartIdentity() *TruncateBuilder { b.restartIdentity = true; return b }
func (b *TruncateBuilder) Cascade() *TruncateBuilder        { b.cascade = true; return b }
func (b *TruncateBuilder) Reason(r string) *TruncateBuilder { b.metaReason = r; return b }

func (b *TruncateBuilder) Build() (string, []any, error) {
	if b.table == "" {
		return "", nil, ErrNoTable
	}
	switch b.d.Name() {
	case "postgres":
		sql := "TRUNCATE TABLE " + b.d.QuoteIdent(b.table)
		if b.restartIdentity {
			sql += " RESTART IDENTITY"
		}
		if b.cascade {
			sql += " CASCADE"
		}
		return sql, nil, nil
	case "mysql", "sqlserver":
		return "TRUNCATE TABLE " + b.d.QuoteIdent(b.table), nil, nil
	case "sqlite":
		return "", nil, ErrTruncateNotSupported
	default:
		return "", nil, ErrTruncateNotSupported
	}
}

func (b *TruncateBuilder) guardMeta() (AuditMeta, int64) {
	return AuditMeta{Op: "TRUNCATE", Table: b.table, Reason: b.metaReason}, -1
}
