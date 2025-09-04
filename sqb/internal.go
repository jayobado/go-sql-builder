package sqb

import "strings"

type buildState struct {
	d    Dialect
	buf  strings.Builder
	args []any
	idx  int // 1-based
}

func (s *buildState) write(sx string) { _, _ = s.buf.WriteString(sx) }

func (s *buildState) emitPredicate(p Pred) {
	for i := 0; i < len(p.sql); i++ {
		if p.sql[i] == '?' {
			s.idx++
			s.write(s.d.Placeholder(s.idx))
		} else {
			s.buf.WriteByte(p.sql[i])
		}
	}
	s.args = append(s.args, p.arg...)
}

func (s *buildState) emitSQL(sql string, args []any) {
	for i := 0; i < len(sql); i++ {
		if sql[i] == '?' {
			s.idx++
			s.write(s.d.Placeholder(s.idx))
		} else {
			s.buf.WriteByte(sql[i])
		}
	}
	s.args = append(s.args, args...)
}

func (s *buildState) result() (string, []any) { return s.buf.String(), s.args }
