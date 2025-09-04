package sqb

type ConflictTarget struct {
	Columns    []string // ON CONFLICT (col1, col2)
	Constraint string   // ON CONSTRAINT name (PG only)
}

func ConflictColumns(cols ...string) ConflictTarget     {
	return ConflictTarget{Columns: cols}
}

func ConflictOnConstraint(name string) ConflictTarget   {
	return ConflictTarget{Constraint: name}
}