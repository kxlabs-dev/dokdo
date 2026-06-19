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

## KX Syntax

### Parameter binding — `#{}`

Replaces with a `?` placeholder and appends the value to the argument slice.

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

### `dokdo.Load(root string) (*Dokdo, error)`

Call once at startup. Parses all `.kx` and `.go` files under `root`. Fails immediately with a descriptive error if a type is missing, unsupported, or unexported.

### `(*Dokdo).Build(target string, params interface{}) (string, []interface{}, error)`

Goroutine-safe. `params` must be a Go struct — maps are rejected. Returns the assembled SQL string and the ordered argument slice.

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

Dokdo returns `(string, []interface{})`. It works with any Go DB library that accepts positional arguments.

| Library | Usage |
|---------|-------|
| `database/sql` | `db.Query(sql, args...)` |
| `sqlx` | `db.Select(&result, sql, args...)` |
| `GORM` | `db.Raw(sql, args...).Scan(&result)` |
| `pgx` | `conn.Query(ctx, sql, args...)` |

---

## License

Dokdo is released under the [MIT License](LICENSE).

The KX Specification that Dokdo implements is licensed under [BSL 1.1](LICENSE-KX). Anyone can use the Dokdo library freely. Reimplementing the KX Specification in a competing product requires a separate license.
