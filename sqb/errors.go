package sqb

import "errors"

var (
	ErrNoTable                    = errors.New("sqb: table not specified")
	ErrNoColumns                  = errors.New("sqb: no columns specified")
	ErrNoRows                     = errors.New("sqb: no rows to insert")
	ErrNoSetClauses               = errors.New("sqb: no SET clauses (nothing to update)")
	ErrJoinWithoutFrom            = errors.New("sqb: JOIN used without FROM")
	ErrNegativeLimit              = errors.New("sqb: LIMIT cannot be negative")
	ErrNegativeOffset             = errors.New("sqb: OFFSET cannot be negative")
	ErrOnConflictNotSupported     = errors.New("sqb: ON CONFLICT not supported by this dialect")
	ErrOnDuplicateNotSupported    = errors.New("sqb: ON DUPLICATE KEY UPDATE not supported by this dialect")
	ErrNoConflictTarget           = errors.New("sqb: ON CONFLICT DO UPDATE requires conflict target")
	ErrConstraintNameNotSupported = errors.New("sqb: constraint-name target unsupported in this dialect")
	ErrNoOpColumnRequired         = errors.New("sqb: MySQL do-nothing upsert requires a no-op column")
	ErrNoMatchColumns             = errors.New("sqb: MERGE requires at least one match column")
	ErrWhereRequired              = errors.New("sqb: WHERE clause required by guard")
	ErrFromRequired               = errors.New("sqb: FROM clause required by guard")
	ErrLimitRequired              = errors.New("sqb: LIMIT required by guard")
	ErrTruncateNotSupported       = errors.New("sqb: TRUNCATE not supported for this dialect")
	ErrNotSQLite                  = errors.New("sqb: operation only valid for SQLite")
)
