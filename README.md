# SQL Builder

A type-safe, multi-dialect SQL query builder for Go with built-in safety guards and audit capabilities.

## Features

- 🏗️ **Fluent Builder API** - Intuitive method chaining for query construction
- 🔒 **Type Safety** - Compile-time safety with proper Go types
- 🎯 **Multi-Dialect Support** - PostgreSQL, MySQL, SQLite, SQL Server
- 🛡️ **Safety Guards** - Prevent dangerous operations like accidental full-table updates/deletes
- 📊 **Audit Trail** - Built-in logging and monitoring of SQL operations
- ⚡ **High Performance** - Efficient SQL generation with minimal allocations
- 🔄 **Advanced Upserts** - Database-specific upsert operations (ON CONFLICT, ON DUPLICATE KEY, MERGE)

## Quick Start

```go
package main

import (
    "context"
    "log"
    
    "github.com/jmoiron/sqlx"
    _ "github.com/lib/pq"
    "github.com/jayobado/sql-builder/sqb"
)

func main() {
    ctx := context.Background()
    d := sqb.Postgres{} // or MySQL{}, SQLite{}, SQLServer{}
    
    db, err := sqlx.Open("postgres", "postgres://user:pass@localhost/db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    // SELECT with WHERE conditions
    sql, args, err := sqb.Select(d).
        Columns("id", "email", "name").
        From("users").
        Where(sqb.And(
            sqb.Eq("status", "active", d),
            sqb.In("role", []string{"admin", "user"}, d),
        )).
        OrderBy("created_at DESC").
        Limit(10).
        Build()
    
    if err != nil {
        log.Fatal(err)
    }
    
    var users []User
    err = db.SelectContext(ctx, &users, sql, args...)
    if err != nil {
        log.Fatal(err)
    }
}
```

## Supported Dialects

### PostgreSQL
```go
d := sqb.Postgres{}
// Features: RETURNING, ON CONFLICT, $1 placeholders, "quoted" identifiers
```

### MySQL
```go
d := sqb.MySQL{}
// Features: ON DUPLICATE KEY UPDATE, ? placeholders, `quoted` identifiers
```

### SQLite
```go
d := sqb.SQLite{
    EnableUpsert:    true,  // Enable ON CONFLICT support
    EnableReturning: true,  // Enable RETURNING support (3.35+)
}
// Features: Optional RETURNING/UPSERT, ? placeholders, "quoted" identifiers
```

### SQL Server
```go
d := sqb.SQLServer{}
// Features: MERGE, OUTPUT, @p1 placeholders, [quoted] identifiers
```

## Query Building

### SELECT Queries

```go
// Basic SELECT
sql, args, err := sqb.Select(d).
    Columns("id", "email", "name").
    From("users").
    Where(sqb.Eq("status", "active", d)).
    Build()
// Result: SELECT "id", "email", "name" FROM "users" WHERE ("status" = $1)

// Complex WHERE with AND/OR
sql, args, err := sqb.Select(d).
    Columns("*").
    From("orders").
    Where(sqb.And(
        sqb.Gte("created_at", "2023-01-01", d),
        sqb.Or(
            sqb.Eq("status", "completed", d),
            sqb.Eq("status", "shipped", d),
        ),
    )).
    OrderBy("created_at DESC").
    Limit(50).
    Build()

// JOINs
sql, args, err := sqb.Select(d).
    Columns("u.name", "p.title").
    From("users u").
    Join("LEFT JOIN posts p ON p.user_id = u.id").
    Where(sqb.NotEq("u.deleted_at", nil, d)).
    Build()
```

### INSERT Queries

```go
// Single row insert
sql, args, err := sqb.Insert(d).
    Into("users").
    Columns("email", "name", "status").
    ValuesRow("alice@example.com", "Alice", "active").
    Build()

// Multi-row insert
sql, args, err := sqb.Insert(d).
    Into("users").
    Columns("email", "name").
    ValuesRows([][]any{
        {"alice@example.com", "Alice"},
        {"bob@example.com", "Bob"},
    }).
    Build()

// PostgreSQL UPSERT
sql, args, err := sqb.Insert(d).
    Into("users").
    Columns("email", "name", "status").
    ValuesRow("alice@example.com", "Alice", "active").
    OnConflictDoUpdate(
        sqb.ConflictColumns("email"),
        sqb.SetExcluded(d, "name", "status"),
    ).
    Returning("id").
    Build()

// MySQL UPSERT
sql, args, err := sqb.Insert(d).
    Into("users").
    Columns("email", "name").
    ValuesRow("alice@example.com", "Alice").
    OnDuplicateKeyUpdate(map[string]sqb.Expr{
        "name": sqb.Values("name", d),
        "updated_at": sqb.RawExpr("NOW()"),
    }).
    Build()
```

### UPDATE Queries

```go
// Basic update
sql, args, err := sqb.Update(d).
    Table("users").
    Set("name", "Alice Updated").
    Set("status", "verified").
    Where(sqb.Eq("id", 1, d)).
    Build()

// Update with expressions
sql, args, err := sqb.Update(d).
    Table("users").
    Set("login_count", 5).
    SetExpr("last_login", sqb.RawExpr("NOW()")).
    Where(sqb.Eq("email", "alice@example.com", d)).
    Returning("id", "updated_at").
    Build()
```

### DELETE Queries

```go
// Basic delete
sql, args, err := sqb.Delete(d).
    From("users").
    Where(sqb.Eq("status", "banned", d)).
    Build()
```

## Safety Guards

Built-in protection against dangerous operations:

```go
// Enable safety guards (enabled by default)
sqb.DefaultSafety.RequireWhereUpdate = true
sqb.DefaultSafety.RequireWhereDelete = true

// This will return ErrWhereRequired
_, _, err := sqb.Update(d).
    Table("users").
    Set("status", "disabled").
    Build() // Error: WHERE clause required by guard

// Explicitly allow full-table operations
sql, args, err := sqb.Update(d).
    Table("temp_cache").
    Set("status", "cleared").
    AllowFullTable().
    Build() // OK

// Allow with row limit
sql, args, err := sqb.Delete(d).
    From("logs").
    AllowFullTableWithMax(1000000).
    Reason("cleanup old logs").
    Build()
```

## Audit Trail

Monitor all SQL operations:

```go
// Set global audit hook
sqb.GlobalAuditHook = func(ctx context.Context, meta sqb.AuditMeta) {
    log.Printf("[AUDIT] %s %s reason=%q dry=%v sql=%s args=%v",
        meta.Op, meta.Table, meta.Reason, meta.DryRun, meta.SQL, meta.Args)
}

// Execute with audit trail
affected, err := sqb.ExecGuardedBuilder(ctx, db,
    sqb.Delete(d).From("old_logs").
        AllowFullTableWithMax(1000000).
        Reason("cleanup old staging data"),
    false, // not a dry run
)
```

## Advanced Features

### Batch Operations

```go
// Large insert with automatic chunking
insert := sqb.Insert(d).
    Into("events").
    Columns("user_id", "event_type", "data")

// Add thousands of rows...
for i := 0; i < 10000; i++ {
    insert.ValuesRow(i, "login", fmt.Sprintf("data_%d", i))
}

// Split into chunks that respect database parameter limits
chunks := insert.MaxParamsChunk(900) // SQLite limit
for _, chunk := range chunks {
    sql, args, err := chunk.Build()
    if err != nil {
        log.Fatal(err)
    }
    _, err = db.ExecContext(ctx, sql, args...)
    if err != nil {
        log.Fatal(err)
    }
}
```

### Struct Integration

```go
type User struct {
    ID     int64  `db:"id"`
    Email  string `db:"email"`
    Name   string `db:"name"`
    Status string `db:"status"`
}

// Extract columns and values from struct
cols, vals, err := sqb.StructColumnsValues(user, "id") // exclude ID
sql, args, err := sqb.Insert(d).
    Into("users").
    Columns(cols...).
    ValuesRow(vals...).
    Build()

// Generate SET map for updates
setMap, err := sqb.StructSetMap(user, "id", "created_at") // exclude ID and created_at
update := sqb.Update(d).Table("users")
for col, val := range setMap {
    update.Set(col, val)
}
sql, args, err := update.Where(sqb.Eq("id", user.ID, d)).Build()
```

### Custom Expressions

```go
// Raw SQL expressions
sql, args, err := sqb.Update(d).
    Table("counters").
    SetExpr("count", sqb.RawExpr("count + ?", 1)).
    SetExpr("updated_at", sqb.RawExpr("NOW()")).
    Where(sqb.Eq("id", 1, d)).
    Build()

// Database functions
sql, args, err := sqb.Select(d).
    ColumnExpr("COUNT(*) as total").
    ColumnExpr("AVG(rating) as avg_rating").
    From("reviews").
    Where(sqb.Gte("created_at", "2023-01-01", d)).
    Build()
```

## Error Handling

The library provides specific error types for common issues:

```go
var (
    ErrNoTable                    = errors.New("sqb: table not specified")
    ErrNoColumns                  = errors.New("sqb: no columns specified")
    ErrNoRows                     = errors.New("sqb: no rows to insert")
    ErrNoSetClauses               = errors.New("sqb: no SET clauses (nothing to update)")
    ErrWhereRequired              = errors.New("sqb: WHERE clause required by guard")
    ErrOnConflictNotSupported     = errors.New("sqb: ON CONFLICT not supported by this dialect")
    // ... and more
)
```

## Performance Tips

1. **Reuse builders**: Create builder instances once and reuse them
2. **Batch operations**: Use `ValuesRows()` for multi-row inserts
3. **Chunk large operations**: Use `MaxParamsChunk()` for very large inserts
4. **Prepared statements**: The generated SQL is prepared-statement friendly

## Testing

```bash
go test ./sqb
```

The library includes comprehensive tests covering:
- All SQL dialects
- Query building correctness
- Safety guard functionality
- Error conditions
- Multi-dialect compatibility

## License

[Add your license here]

## Contributing

[Add contributing guidelines here]
