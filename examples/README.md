# SQL Builder Examples

This directory contains examples demonstrating how to use the SQL Builder library.

## Directory Structure

- `bulder/` - Contains examples for the SQL builder functionality
- `user/` - Contains user-related examples

## Getting Started

Each subdirectory contains its own examples and documentation. Navigate to the specific directory to find relevant code samples and usage patterns.

## Common SQL Builder Patterns

### Basic Query Building

```go
// Simple SELECT query
sql, args, err := sqb.Select(dialect).
    Columns("id", "name", "email").
    From("users").
    Where(sqb.Eq("active", true, dialect)).
    Build()

// INSERT with returning
sql, args, err := sqb.Insert(dialect).
    Into("users").
    Columns("name", "email", "active").
    ValuesRow("John Doe", "john@example.com", true).
    Returning("id").
    Build()
```

### Advanced Features

- **Type Safety**: Compile-time SQL validation
- **Multi-Dialect Support**: PostgreSQL, MySQL, SQLite, SQL Server
- **Safety Guards**: Prevent dangerous operations
- **Audit Trails**: Track database changes
- **Parameter Binding**: Automatic SQL injection prevention

## Running Examples

Navigate to the specific example directories and run:

```bash
go run *.go
```

Each example includes detailed comments explaining the SQL Builder features being demonstrated.
