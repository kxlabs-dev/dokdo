# Dok(trin)do

**Dynamic SQL builder for Go — powered by KX template syntax.**

Dokdo assembles dynamic SQL from `.kx` template files and validates parameter types at load time using `go/ast`. It returns a SQL string and a bound argument slice. DB connection, query execution, and result mapping are yours to choose.

```go
dq, err := dokdo.Load("query/")

sql, args, err := dq.Build("users#selectUser", UserParams{Name: &name})
// sql  → "SELECT * FROM users WHERE name = ?"
// args → []interface{}{"kim"}

rows, err := db.Query(sql, args...)     // database/sql
db.Raw(sql, args...).Scan(&result)      // GORM
db.Select(&result, sql, args...)        // sqlx
conn.Query(ctx, sql, args...)           // pgx
```

`?` above is the default (MySQL/MariaDB) placeholder. Dokdo also supports PostgreSQL, Oracle, and SQL Server — see [Dialects](#dialects).

---

## Why Dok(trin)do?

Go's dynamic query story is fragmented. String concatenation is unsafe. ORMs hide SQL. Query builders scatter logic across method chains.

Dokdo is a declaration-driven SQL builder — execute only what is declared, nothing more. SQL stays in `.kx` files, easier and more flexible than MyBatis. Parameter structs are validated at startup. SQL injection is blocked structurally, not after the fact.

---

## Installation

```bash
go get github.com/kxlabs-dev/dokdo
```

Requires Go 1.22+

---

## Quick Start

**1. Declare your parameter type**

```go
// query/users.go
// package declaration is required. The package name can be anything.
package query

type UserParams struct {
    Name   *string
    Score  *int
    Status *string
}
```

**2. Write your query**

The root tag name must match the filename (without extension). A mismatch results in a `ParseError`.

```kx
<!-- query/users.kx → root tag must be <users> -->
<users>
  <selectUser set:{"users#UserParams"}>
    SELECT * FROM users
    <where>
      [[ if name != nil :{
        AND name = #{name}
      }]]
      [[ if score != nil :{
        AND score \> #{score}
      }]]
      [[ if status != nil :{
        AND status = #{status}
      }]]
    </>
  </>
</>
```

**3. Load and build**

```go
dq, err := dokdo.Load("query/")
if err != nil {
    log.Fatal(err)
}

name := "kim"
sql, args, err := dq.Build("users#selectUser", UserParams{Name: &name})
```

---

## Dialects

`Load()` accepts an optional dialect argument controlling the placeholder format. Omitting it defaults to MySQL — existing `Load("query/")` calls keep working unchanged.

```go
dq, err := dokdo.Load("query/")                              // ? — MySQL/MariaDB (default)
dq, err := dokdo.Load("query/", dokdo.DialectMySQL)           // ? — explicit
dq, err := dokdo.Load("query/", dokdo.DialectPostgres)        // $1, $2, $3...
dq, err := dokdo.Load("query/", dokdo.DialectOracle)          // :1, :2, :3...
dq, err := dokdo.Load("query/", dokdo.DialectSQLServer)       // @p1, @p2, @p3...
```

The `args` slice returned by `Build()` stays positional regardless of dialect — all four drivers (`database/sql`, `pgx`, `go-ora`, `go-mssqldb`) accept ordinal argument slices, so no named-parameter wrapping is needed.

Only the placeholder format is translated. SQL grammar differences across databases (`LIMIT`/`TOP`, `AUTO_INCREMENT`/`SERIAL`/`IDENTITY`, etc.) are out of scope — Dokdo is not an ORM and never rewrites your SQL text.

### Usage by database

Each example builds the same query and runs it through the driver typically used with that database. `database/sql` examples assume the matching driver is already imported for its side effects (e.g. `_ "github.com/go-sql-driver/mysql"`).

**MySQL / MariaDB (go-sql-driver/mysql)**

```go
dq, _ := dokdo.Load("query/")   // DialectMySQL is the default

sql, args, _ := dq.Build("users#selectUser", UserParams{Name: &name})
rows, err := db.Query(sql, args...)   // db is *sql.DB opened with go-sql-driver/mysql
```

**PostgreSQL (pgx)**

```go
dq, _ := dokdo.Load("query/", dokdo.DialectPostgres)

sql, args, _ := dq.Build("users#selectUser", UserParams{Name: &name})
rows, err := db.Query(sql, args...)   // db is *sql.DB opened with pgx/stdlib, or a pgx.Conn
```

**Oracle (go-ora)**

```go
dq, _ := dokdo.Load("query/", dokdo.DialectOracle)

sql, args, _ := dq.Build("users#selectUser", UserParams{Name: &name})
rows, err := db.Query(sql, args...)   // db is *sql.DB opened with sijms/go-ora
```

**SQL Server (go-mssqldb)**

```go
dq, _ := dokdo.Load("query/", dokdo.DialectSQLServer)

sql, args, _ := dq.Build("users#selectUser", UserParams{Name: &name})
rows, err := db.Query(sql, args...)   // db is *sql.DB opened with microsoft/go-mssqldb
```

**GORM (gorm.io/gorm)**

GORM works the same way for MySQL, PostgreSQL, and Oracle:

```go
dq, _ := dokdo.Load("query/", dokdo.DialectPostgres)   // or DialectMySQL / DialectOracle

sql, args, _ := dq.Build("users#selectUser", UserParams{Name: &name})
db.Raw(sql, args...).Scan(&result)
```

SQL Server is the one exception — load with `DialectMySQL` instead of `DialectSQLServer` when using GORM. See [Compatibility](#compatibility) for why.

```go
dq, _ := dokdo.Load("query/", dokdo.DialectMySQL)   // not DialectSQLServer

sql, args, _ := dq.Build("users#selectUser", UserParams{Name: &name})
db.Raw(sql, args...).Scan(&result)   // db is a *gorm.DB opened with gorm.io/driver/sqlserver
```

---

## KX Syntax

### Parameter binding — `#{}`

Replaces with a placeholder (`?`, `$N`, `:N`, or `@pN` depending on dialect) and appends the value to the argument slice.

```kx
AND name = #{name}
AND score > #{score}
AND city = #{user.address.city}
```

Dot notation resolves nested struct fields.

### Identifier insertion — `${}`

Inserts a raw identifier (column name, table alias, etc.) directly into the SQL. Only allowed inside `for` loops. Validated against an allowlist and blocklist — SQL injection is blocked structurally.

```kx
[[ for col in columns :{
  ${col},
}]]
```

### `<where>` tag

Wraps conditions in a `WHERE` clause. Omits `WHERE` entirely when all conditions are nil. Strips the leading `AND` / `OR` automatically.

```kx
<where>
  [[ if name != nil :{
    AND name = #{name}
  }]]
  [[ if status != nil :{
    AND status = #{status}
  }]]
</>
```

### `[[ if ]]`

```kx
[[ if score > 80 :{
  AND grade = 'A'
} else if score > 60 :{
  AND grade = 'B'
} else :{
  AND grade = 'C'
}]]
```

### `[[ switch ]]`

```kx
[[ switch (status) :{
  case ("VIP") :{
    AND grade = 'A'
  }
  default :{
    AND grade = 'C'
  }
}]]
```

### `[[ for ]]`

**Scalar list — IN clause:**

```go
type IdParams struct {
    IdList []int64
}
```

```kx
AND id IN (
  [[ for id in idList :{
    #{id},
  }]]
)
```

**Column list:**

```go
type ColParams struct {
    Columns []string
}
```

```kx
SELECT
[[ for col in columns :{
  ${col},
}]]
FROM users
```

**Anonymous struct slice — dynamic SET / WHERE:**

```go
type UpdateParams struct {
    Id      int64
    Updates []struct {
        Key   string
        Value string
    }
}
```

```kx
UPDATE users SET
[[ for field in updates :{
  ${field.Key} = #{field.Value},
}]]
WHERE id = #{id}
```

The trailing `,` after the last item is removed automatically. `UNION` / `UNION ALL` handling is left to the developer.

### Nesting

KX prohibits `[[ ]]` nesting by design. Dokdo is a SQL builder — there are no components to split into and no JS to preprocess data — so nesting is permitted as a purposeful exception for this context.

```kx
[[ if orders != nil :{
  AND order_id IN (
    [[ for order in orders :{
      #{order.Id},
    }]]
  )
}]]
```

For the full KX language specification, see [KX Specification](https://github.com/luna-kx/kx-spec).

### Escape sequences

| Write | Outputs |
|-------|---------|
| `\>` | `>` |
| `\<` | `<` |
| `!=` | `<>` |
| `\\` | `\` |

Comparison operators inside `[[ ]]` conditions need no escaping.

---

## Type System

Declare parameter types in `.go` files in the same directory as your `.kx` files. A `package` declaration is required by Go syntax — the package name does not matter. Dokdo parses them with `go/ast` at `Load()` time and validates parameter structs at `Build()` time — before any SQL reaches the DB.

**Supported field types:**

| Type | Notes |
|------|-------|
| `int`, `int8` … `int64` | Pointer allowed: `*int64` |
| `uint`, `uint8` … `uint64` | Pointer allowed |
| `float32`, `float64` | Pointer allowed |
| `string` | Pointer allowed: `*string` |
| `bool` | Pointer allowed |
| `[]int`, `[]int64`, `[]float64`, `[]string` | Scalar slices |
| `[]struct{ Field Type; ... }` | Anonymous struct slice |

**Forbidden types → `BuildError` at `Load()`:**

```go
// Not allowed
type Bad struct {
    Data    map[string]interface{}  // BuildError
    Updates []CustomType            // BuildError
}

// Use this instead
type Good struct {
    Updates []struct {
        Key   string
        Value string
    }
}
```

Pointer fields (`*T`) are nullable. Non-pointer fields are required.

---

## Project Layout

```
query/
  users.kx       ← SQL templates
  users.go       ← parameter type declarations
  orders.kx
  orders.go
  users/
    detail.kx    ← subdirectories supported
    detail.go
```

The root tag in each `.kx` file must match the filename. Queries are addressed as `filename#queryName` (or `dir/filename#queryName` for subdirectories).

---

## API

### `dokdo.Load(root string, dialect ...Dialect) (*Dokdo, error)`

Call once at startup. Parses all `.kx` and `.go` files under `root`. Fails immediately with a descriptive error if a type is missing, unsupported, or unexported.

`dialect` is optional and defaults to `DialectMySQL`. If multiple values are passed, the last one wins (standard Go variadic-options behavior — no error is raised). See [Dialects](#dialects).

### `(*Dokdo).Build(target string, params interface{}) (string, []interface{}, error)`

Goroutine-safe. `params` must be a Go struct — maps are rejected. Returns the assembled SQL string and the ordered argument slice, using the placeholder format of the dialect the `*Dokdo` instance was loaded with.

---

## Error Reference

| Error | When |
|-------|------|
| `ParseError` | Invalid `.kx` syntax |
| `BuildError` | Missing type, unexported type, forbidden field type |
| `TypeMismatchError` | Struct field type does not match declaration |
| `RequiredFieldError` | Non-pointer field is nil |
| `TagNotFoundError` | Target query not found |
| `InvalidParamsError` | `params` is a map |
| `RuntimeError` | `${}` identifier fails injection validation |

---

## Compatibility

Dokdo returns `(string, []interface{})`. It works with any Go DB library that accepts positional arguments. Verified end-to-end (real database round-trip, CRUD) against MySQL/MariaDB, PostgreSQL, Oracle, and SQL Server, through both raw drivers and GORM.

| Library | Usage |
|---------|-------|
| `database/sql` | `db.Query(sql, args...)` |
| `sqlx` | `db.Select(&result, sql, args...)` |
| `GORM` | `db.Raw(sql, args...).Scan(&result)` |
| `pgx` | `conn.Query(ctx, sql, args...)` |

**GORM + SQL Server:** load with `dokdo.DialectMySQL` (`?`), not `DialectSQLServer`. GORM's `Raw()`/`Exec()` routes any SQL containing `@` through its named-parameter path, which doesn't recognize positional `@p1`-style placeholders and silently drops the bound arguments. `go-mssqldb` accepts `?` directly, and GORM's `?` path binds arguments correctly — `DialectMySQL` is the working choice for GORM users on SQL Server. Using `database/sql`/`go-mssqldb` directly, `DialectSQLServer` works as expected.

---

## License

Dokdo is released under the [MIT License](LICENSE).

The KX Specification that Dokdo implements is licensed under [BSL 1.1](LICENSE-KX). Anyone can use the Dokdo library freely. Reimplementing the KX Specification in a competing product requires a separate license.
