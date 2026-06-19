# Dokdo

**Go를 위한 동적 SQL 빌더 — KX 템플릿 문법 기반**

Dokdo는 `.kx` 템플릿 파일에서 동적 SQL을 조립하고, `go/ast`를 사용해 로드 시점에 파라미터 타입을 검증합니다. SQL 문자열과 바인딩 파라미터 슬라이스를 반환합니다. DB 연결, 쿼리 실행, 결과 매핑은 개발자가 선택한 라이브러리가 담당합니다.

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

## 왜 Dokdo인가

Go의 동적 쿼리 생태계는 파편화되어 있습니다. 문자열 이어붙이기는 위험하고, ORM은 SQL을 숨기고, 쿼리 빌더는 로직을 메서드 체인으로 분산시킵니다. Go를 위한 MyBatis가 없었습니다 — 지금까지는.

Dokdo는 SQL을 `.kx` 파일에서 관리하고, 앱 시작 시 파라미터 struct를 검증하며, SQL 인젝션을 구조적으로 차단합니다.

---

## 설치

```bash
go get github.com/luna-kx/dokdo
```

Go 1.21 이상 필요

---

## 빠른 시작

**1. 파라미터 타입 선언**

```go
// query/users.go
// package 선언은 필수. 패키지명은 무엇이든 상관없습니다.
package query

type UserParams struct {
    Name   *string
    Score  *int
    Status *string
}
```

**2. 쿼리 작성**

최상위 태그명은 파일명(확장자 제외)과 반드시 일치해야 합니다. 불일치 시 `ParseError`.

```kx
<!-- query/users.kx → 최상위 태그는 반드시 <users> -->
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

**3. 로드 후 빌드**

```go
dq, err := dokdo.Load("query/")
if err != nil {
    log.Fatal(err)
}

name := "kim"
sql, args, err := dq.Build("users#selectUser", UserParams{Name: &name})
```

---

## KX 문법

### 바인딩 파라미터 — `#{}`

`?` 플레이스홀더로 치환하고 값을 파라미터 슬라이스에 추가합니다.

```kx
AND name = #{name}
AND score > #{score}
AND city = #{user.address.city}
```

점 표기법으로 중첩 struct 필드에 접근합니다.

### 식별자 직접 삽입 — `${}`

컬럼명, 테이블 별칭 등 식별자를 직접 삽입합니다. `for` 문 안에서만 사용 가능합니다. 허용목록과 차단목록으로 SQL 인젝션을 구조적으로 차단합니다.

```kx
[[ for col in columns :{
  ${col},
}]]
```

### `<where>` 태그

조건을 `WHERE` 절로 감쌉니다. 모든 조건이 nil이면 `WHERE` 키워드 자체를 생략합니다. 첫 번째 `AND` / `OR`를 자동으로 제거합니다.

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

**스칼라 리스트 — IN 절:**

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

**컬럼 나열:**

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

**익명 struct 슬라이스 — 동적 SET / WHERE:**

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

for 루프 마지막 `,` 는 자동 제거됩니다. `UNION` / `UNION ALL` 처리는 개발자가 직접 설계합니다.

### 중첩

KX 기본 문법에서 `[[ ]]` 중첩은 금지입니다. Dokdo는 SQL 빌더라는 특성상 — 컴포넌트로 분리하거나 JS로 가공할 수 없는 환경이므로 — 예외적으로 `[[ ]]` 중첩을 허용합니다.

```kx
[[ if orders != nil :{
  AND order_id IN (
    [[ for order in orders :{
      #{order.Id},
    }]]
  )
}]]
```

KX 언어 전체 스펙은 [KX Specification](https://github.com/luna-kx/kx-spec)을 참조하세요.

### 이스케이프

| 표기 | SQL 출력 |
|------|---------|
| `\>` | `>` |
| `\<` | `<` |
| `!=` | `<>` |
| `\\` | `\` |

`[[ ]]` 조건식 안의 비교 연산자는 이스케이프 불필요합니다.

---

## 타입 시스템

파라미터 타입을 `.kx` 파일과 같은 디렉토리의 `.go` 파일에 선언합니다. `package` 선언은 Go 문법상 필수이며 패키지명은 무엇이든 상관없습니다. Dokdo가 `Load()` 시점에 `go/ast`로 파싱하고, `Build()` 시점에 파라미터 struct를 검증합니다. SQL이 DB에 도달하기 전에 타입 불일치를 잡습니다.

**지원 필드 타입:**

| 타입 | 비고 |
|------|------|
| `int`, `int8` … `int64` | 포인터 허용: `*int64` |
| `uint`, `uint8` … `uint64` | 포인터 허용 |
| `float32`, `float64` | 포인터 허용 |
| `string` | 포인터 허용: `*string` |
| `bool` | 포인터 허용 |
| `[]int`, `[]int64`, `[]float64`, `[]string` | 스칼라 슬라이스 |
| `[]struct{ 필드 타입; ... }` | 익명 struct 슬라이스 |

**금지 타입 → `Load()` 시 `BuildError`:**

```go
// 금지
type Bad struct {
    Data    map[string]interface{}  // BuildError
    Updates []CustomType            // BuildError
}

// 허용
type Good struct {
    Updates []struct {
        Key   string
        Value string
    }
}
```

포인터 필드(`*T`)는 nil 허용입니다. 비포인터 필드는 required입니다.

---

## 프로젝트 구조

```
query/
  users.kx       ← SQL 템플릿
  users.go       ← 파라미터 타입 선언
  orders.kx
  orders.go
  users/
    detail.kx    ← 서브디렉토리 지원
    detail.go
```

각 `.kx` 파일의 최상위 태그는 파일명과 일치해야 합니다. 쿼리는 `파일명#쿼리명` (서브디렉토리: `디렉토리/파일명#쿼리명`) 형식으로 지정합니다.

---

## API

### `dokdo.Load(root string) (*Dokdo, error)`

앱 시작 시 1회 호출합니다. `root` 하위의 모든 `.kx`와 `.go` 파일을 파싱합니다. 타입이 없거나, 지원하지 않거나, unexported이면 즉시 에러를 반환합니다.

### `(*Dokdo).Build(target string, params interface{}) (string, []interface{}, error)`

goroutine-safe. `params`는 Go struct여야 합니다. map은 거부됩니다. 조립된 SQL 문자열과 순서 기반 파라미터 슬라이스를 반환합니다.

---

## 에러

| 에러 | 발생 시점 |
|------|---------|
| `ParseError` | `.kx` 파일 문법 오류 |
| `BuildError` | 타입 미존재, unexported 타입, 금지 필드 타입 |
| `TypeMismatchError` | struct 필드 타입 불일치 |
| `RequiredFieldError` | 비포인터 필드가 nil |
| `TagNotFoundError` | target 쿼리 미존재 |
| `InvalidParamsError` | params가 map 타입 |
| `RuntimeError` | `${}` 식별자 인젝션 검증 실패 |

---

## 호환성

Dokdo는 `(string, []interface{})`를 반환합니다. 순서 기반 파라미터를 지원하는 모든 Go DB 라이브러리와 호환됩니다.

| 라이브러리 | 사용 방법 |
|-----------|---------|
| `database/sql` | `db.Query(sql, args...)` |
| `sqlx` | `db.Select(&result, sql, args...)` |
| `GORM` | `db.Raw(sql, args...).Scan(&result)` |
| `pgx` | `conn.Query(ctx, sql, args...)` |

---

## 라이선스

Dokdo 코드는 [MIT 라이선스](LICENSE)로 배포됩니다.

Dokdo가 구현하는 KX 스펙은 [BSL 1.1](LICENSE-KX)을 따릅니다. Dokdo 라이브러리는 누구나 자유롭게 사용할 수 있습니다. KX 스펙을 기반으로 경쟁 구현체를 만들려면 별도 라이선스가 필요합니다.
