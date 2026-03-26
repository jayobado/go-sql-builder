# SQL Builder

A complete SQL builder for Go with type-safe DDL/DML operations and multi-dialect support.

## Features

- 🏗️ **Complete SQL Support** - DDL (CREATE, ALTER, DROP) + DML (SELECT, INSERT, UPDATE, DELETE)
- 🔒 **Type-Safe Schema** - Portable data types with dialect-specific mappings
- 🎯 **Multi-Dialect** - PostgreSQL, MySQL, SQLite, SQL Server with proper quoting and placeholders
- 🛡️ **Safety Guards** - Prevent dangerous operations with configurable protections
- � **Advanced Upserts** - Native upsert support (ON CONFLICT, ON DUPLICATE KEY, MERGE)
- 📊 **Schema Migration** - ALTER TABLE operations with dialect-aware handling
- ⚡ **High Performance** - Efficient SQL generation with minimal allocations

## Quick Start

```go
package main

import (
    "log"
    "github.com/jayobado/go-sql-builder/sqb"
)

func main() {
    d := sqb.Postgres{} // or MySQL{}, SQLite{}, SQLServer{}

    // CREATE TABLE with type-safe columns
    sql, _, err := sqb.CreateTable(d).
        Table("users").
        IfNotExists().
        Column("id", sqb.BigInt(), sqb.AutoIncrement(), sqb.NotNull()).
        Column("email", sqb.Varchar(255), sqb.NotNull()).
        Column("created_at", sqb.Timestamptz(), sqb.DefaultLiteral("NOW()")).
        PrimaryKey("id").
        UniqueIndex("idx_users_email", "email").
        Build()

    // SELECT with complex WHERE
    sql, args, err := sqb.Select(d).
        Columns("id", "email").
        From("users").
        Where(sqb.And(
            sqb.Eq("status", "active", d),
            sqb.In("role", []string{"admin", "user"}, d),
        )).
        OrderBy("created_at DESC").
        Limit(10).
        Build()
    // Result: SELECT "id", "email" FROM "users" WHERE (("status" = $1) AND ("role" IN ($2,$3))) ORDER BY created_at DESC LIMIT 10

    // UPSERT with proper conflict handling
    sql, args, err = sqb.Insert(d).
        Into("users").
        Columns("email", "name").
        ValuesRow("alice@example.com", "Alice").
        OnConflictDoUpdate(
            sqb.ConflictColumns("email"),
            sqb.SetExcluded(d, "name"),
        ).
        Returning("id").
        Build()
}
```

## Type System

The library provides portable data types that map correctly across dialects:

```go
// Basic types
sqb.Text()                    // TEXT, NVARCHAR(MAX), etc.
sqb.Varchar(255)             // VARCHAR(255), TEXT (SQLite)
sqb.Integer()                // INTEGER, INT
sqb.BigInt()                 // BIGINT
sqb.Boolean()                // BOOLEAN, TINYINT(1), BIT
sqb.Decimal(10, 2)           // DECIMAL(10,2)

// Date/time types
sqb.Date()                   // DATE
sqb.Timestamp()              // TIMESTAMP, DATETIME, DATETIME2
sqb.Timestamptz()            // TIMESTAMPTZ, DATETIMEOFFSET

// JSON and specialized types
sqb.JSON()                   // JSON, NVARCHAR(MAX)
sqb.JSONB()                  // JSONB (PostgreSQL)
sqb.UUID()                   // UUID, CHAR(36), UNIQUEIDENTIFIER
sqb.Binary(16)               // BINARY(16), BLOB

// Dialect-specific enums
sqb.EnumPG("status_type")    // PostgreSQL enum reference
sqb.EnumMySQL("small", "large") // MySQL inline enum
```

## DDL Operations

### CREATE TABLE

```go
sql, _, err := sqb.CreateTable(d).
    Table("orders").
    IfNotExists().
    Column("id", sqb.UUID(), sqb.UUIDPrimaryKey(d)).
    Column("user_id", sqb.BigInt(), sqb.NotNull()).
    Column("amount", sqb.Decimal(10, 2), sqb.NotNull()).
    Column("status", sqb.Varchar(20), sqb.DefaultLiteral("'pending'")).
    Column("created_at", sqb.Timestamptz(), sqb.TimestamptzNow(d)).
    Column("metadata", sqb.JSONB()).
    PrimaryKey("id").
    ForeignKey([]string{"user_id"}, "users", []string{"id"}, "CASCADE", "").
    UniqueIndex("idx_orders_user_id", "user_id", "created_at").
    Build()
```

### ALTER TABLE

```go
// Multiple operations in one call
stmts, err := sqb.AlterTable(d).
    Table("users").
    AddColumn("nickname", sqb.Varchar(100)).
    RenameColumn("name", "full_name", sqb.Varchar(255)).  // MySQL needs type
    SetDefault("status", "'active'").
    AddUnique("uq_users_email", "email").
    BuildMany()

// Individual operations
sql, _, err := sqb.AlterTable(d).
    Table("products").
    AlterType("price", sqb.Decimal(12, 4)).
    Build()
```

## DML Operations

### SELECT Queries

```go
// Complex SELECT with joins and aggregations
sql, args, err := sqb.Select(d).
    Columns("u.id", "u.email").
    ColumnExpr("COUNT(o.id) as order_count").
    From("users u").
    Join("LEFT JOIN orders o ON o.user_id = u.id").
    Where(sqb.And(
        sqb.Gte("u.created_at", "2023-01-01", d),
        sqb.Or(
            sqb.Eq("u.status", "active", d),
            sqb.Eq("u.status", "premium", d),
        ),
    )).
    GroupBy("u.id", "u.email").
    Having(sqb.Gt("COUNT(o.id)", 5, d)).
    OrderBy("order_count DESC").
    Limit(20).
    Build()
```

### Dialect-Specific Upserts

```go
// PostgreSQL: ON CONFLICT with conditional update
sql, args, err := sqb.Insert(d).
    Into("users").
    Columns("email", "name", "status").
    ValuesRow("alice@example.com", "Alice", "active").
    OnConflictDoUpdate(
        sqb.ConflictColumns("email"),
        sqb.SetExcluded(d, "name"),  // Only update name, not status
    ).
    OnConflictWhere(sqb.NotEq("status", "banned", d)).  // Skip if banned
    Returning("id").
    Build()

// MySQL: ON DUPLICATE KEY UPDATE with VALUES()
sql, args, err := sqb.Insert(d).
    Into("counters").
    Columns("key", "value").
    ValuesRow("page_views", 1).
    OnDuplicateKeyUpdate(map[string]sqb.Expr{
        "value": sqb.RawExpr("value + VALUES(value)"),  // Increment
        "updated_at": sqb.RawExpr("NOW()"),
    }).
    Build()

// SQL Server: MERGE statement
sql, args, err := sqb.Insert(d).
    Into("inventory").
    Columns("product_id", "quantity").
    ValuesRow(123, 50).
    MSSQLMergeOn([]string{"product_id"}, map[string]sqb.Expr{
        "quantity": sqb.RawExpr("s.quantity"),
    }).
    OutputInserted("product_id", "quantity").
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

### Expressions and Functions

```go
// UUID generation (dialect-aware)
insert := sqb.Insert(d).
    Into("users").
    Columns("id", "email", "created_at").
    ValuesRow(sqb.UUIDv4(d), "user@example.com", sqb.RawExpr("NOW()"))

// Complex expressions
sql, args, err := sqb.Update(d).
    Table("stats").
    SetExpr("score", sqb.RawExpr("GREATEST(score, ?)", 100)).
    SetExpr("last_updated", sqb.RawExpr("NOW()")).
    Where(sqb.Eq("user_id", 123, d)).
    Build()
```

### Batch Operations

```go
// Large batch with automatic chunking
insert := sqb.Insert(d).Into("events").Columns("user_id", "event_type")
for i := 0; i < 10000; i++ {
    insert.ValuesRow(i, "login")
}

// Split into dialect-appropriate chunks
chunks := insert.MaxParamsChunk(900) // SQLite parameter limit
for _, chunk := range chunks {
    sql, args, err := chunk.Build()
    // Execute each chunk...
}
```

## Supported Dialects

| Feature            | PostgreSQL    | MySQL              | SQLite          | SQL Server      |
| ------------------ | ------------- | ------------------ | --------------- | --------------- |
| **Identifiers**    | `"quoted"`    | `` `quoted` ``     | `"quoted"`      | `[quoted]`      |
| **Placeholders**   | `$1, $2`      | `?, ?`             | `?, ?`          | `@p1, @p2`      |
| **RETURNING**      | ✅            | ❌                 | ✅ (3.35+)      | ❌              |
| **Upserts**        | `ON CONFLICT` | `ON DUPLICATE KEY` | `ON CONFLICT`   | `MERGE`         |
| **JSON**           | `JSON/JSONB`  | `JSON`             | `JSON`          | `NVARCHAR(MAX)` |
| **AUTO_INCREMENT** | `IDENTITY`    | `AUTO_INCREMENT`   | `AUTOINCREMENT` | `IDENTITY(1,1)` |
| **ALTER TABLE**    | Full support  | Full support       | Limited         | Full support    |

```go
// Configure SQLite features
d := sqb.SQLite{
    EnableUpsert:    true,  // ON CONFLICT support (3.24+)
    EnableReturning: true,  // RETURNING support (3.35+)
}
```

## Testing

The library includes comprehensive tests covering all dialects and features:

```bash
go test ./sqb -v
```

All tests pass and cover:

- SQL generation correctness across dialects
- Type system mappings
- Safety guard functionality
- Complex query scenarios
- Error conditions

## Performance & Best Practices

1. **Reuse builders** for repeated operations
2. **Use typed columns** for better portability
3. **Leverage batch operations** for large datasets
4. **Enable safety guards** in production
5. **Use prepared statements** - all SQL is prepared-statement ready
