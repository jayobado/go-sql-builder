package sqb

type IndexColOption func(*IndexPart)

func Desc() IndexColOption       { return func(p *IndexPart) { p.desc = true } }
func Asc() IndexColOption        { return func(p *IndexPart) { p.desc = false } }
func NullsFirst() IndexColOption { return func(p *IndexPart) { p.nulls = "FIRST" } }
func NullsLast() IndexColOption  { return func(p *IndexPart) { p.nulls = "LAST" } }
func Length(n int) IndexColOption {
	return func(p *IndexPart) { if n > 0 { p.length = n } }
}
func Collate(name string) IndexColOption {
	return func(p *IndexPart) { p.collate = name }
}
