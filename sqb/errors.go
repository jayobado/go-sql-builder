package sqb

import "errors"

var (
	// Basic validation errors
	ErrNoTable      = errors.New("sqb: table not specified")
	ErrNoColumns    = errors.New("sqb: no columns specified")
	ErrNoRows       = errors.New("sqb: no rows to insert")
	ErrNoSetClauses = errors.New("sqb: no SET clauses (nothing to update)")

	// Query structure errors
	ErrJoinWithoutFrom = errors.New("sqb: JOIN used without FROM")
	ErrNegativeLimit   = errors.New("sqb: LIMIT cannot be negative")
	ErrNegativeOffset  = errors.New("sqb: OFFSET cannot be negative")

	// Safety guard errors
	ErrWhereRequired = errors.New("sqb: WHERE clause required by guard")
	ErrFromRequired  = errors.New("sqb: FROM clause required by guard")
	ErrLimitRequired = errors.New("sqb: LIMIT required by guard")

	// Upsert/conflict resolution errors
	ErrOnConflictNotSupported     = errors.New("sqb: ON CONFLICT not supported by this dialect")
	ErrOnDuplicateNotSupported    = errors.New("sqb: ON DUPLICATE KEY UPDATE not supported by this dialect")
	ErrNoConflictTarget           = errors.New("sqb: ON CONFLICT DO UPDATE requires conflict target")
	ErrConstraintNameNotSupported = errors.New("sqb: constraint name not supported by this dialect")
	ErrNoOpColumnRequired         = errors.New("sqb: MySQL do-nothing upsert requires a no-op column")
	ErrNoMatchColumns             = errors.New("sqb: MERGE requires at least one match column")

	// Dialect-specific errors
	ErrTruncateNotSupported   = errors.New("sqb: TRUNCATE not supported for this dialect")
	ErrNotSQLite              = errors.New("sqb: operation only valid for SQLite")
	ErrAlterOpNotSupported    = errors.New("sqb: alter operation not supported by this dialect")
	ErrConstraintNameRequired = errors.New("sqb: constraint name is required")
	ErrMySQLTypeRequired      = errors.New("sqb: MySQL requires a column type for this operation")

	// Index-related errors
	ErrIndexWhereNotSupported   = errors.New("sqb: index WHERE/filtered indexes not supported by this dialect")
	ErrIndexIncludeNotSupported = errors.New("sqb: index INCLUDE not supported by this dialect")
	ErrIndexTableRequired       = errors.New("sqb: index drop requires table name for this dialect")
)
