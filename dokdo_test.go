package dokdo_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kxlabs-dev/dokdo"
)

func writeKX(t *testing.T, dir, relPath, content string) {
	t.Helper()
	full := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile %s: %v", relPath, err)
	}
}

func writeGo(t *testing.T, dir, relPath, content string) {
	t.Helper()
	full := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile %s: %v", relPath, err)
	}
}

func mustLoad(t *testing.T, dir string) *dokdo.Dokdo {
	t.Helper()
	dq, err := dokdo.Load(dir)
	if err != nil {
		t.Fatalf("Load(%q): %v", dir, err)
	}
	return dq
}

// ─────────────────────────────────────────
// 정상 케이스
// ─────────────────────────────────────────

func TestBasicSelect(t *testing.T) {
	dir, err := os.MkdirTemp("", "dokdo-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	writeGo(t, dir, "users.go", `
package query

type UserParams struct {
	Name *string
}
`)
	writeKX(t, dir, "users.kx", `
<users>
  <selectUser set:{"users#UserParams"}>
    SELECT * FROM users
    WHERE name = #{name}
  </>
</>
`)

	dq := mustLoad(t, dir)

	type Params struct{ Name *string }
	name := "kim"
	sql, args, err := dq.Build("users#selectUser", Params{Name: &name})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !strings.Contains(sql, "?") {
		t.Errorf("sql should contain '?', got: %q", sql)
	}
	if len(args) == 0 {
		t.Fatal("args should not be empty")
	}
	ptr, ok := args[0].(*string)
	if !ok {
		t.Fatalf("args[0] type: got %T, want *string", args[0])
	}
	if *ptr != name {
		t.Errorf("args[0] value: got %q, want %q", *ptr, name)
	}
}

func TestWhereTagAllNil(t *testing.T) {
	dir, err := os.MkdirTemp("", "dokdo-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	writeGo(t, dir, "users.go", `
package query

type UserParams struct {
	Name *string
}
`)
	writeKX(t, dir, "users.kx", `
<users>
  <selectUser set:{"users#UserParams"}>
    SELECT * FROM users
    <where>
      [[ if name != nil :{
        AND name = #{name}
      }]]
    </>
  </>
</>
`)
	dq := mustLoad(t, dir)

	type Params struct{ Name *string }
	sql, _, err := dq.Build("users#selectUser", Params{})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if strings.Contains(sql, "WHERE") {
		t.Errorf("WHERE should be absent when all conditions are nil, got: %q", sql)
	}
}

func TestWhereTagWithCondition(t *testing.T) {
	dir, err := os.MkdirTemp("", "dokdo-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	writeGo(t, dir, "users.go", `
package query

type UserParams struct {
	Name *string
}
`)
	writeKX(t, dir, "users.kx", `
<users>
  <selectUser set:{"users#UserParams"}>
    SELECT * FROM users
    <where>
      [[ if name != nil :{
        AND name = #{name}
      }]]
    </>
  </>
</>
`)
	dq := mustLoad(t, dir)

	type Params struct{ Name *string }
	name := "kim"
	sql, _, err := dq.Build("users#selectUser", Params{Name: &name})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !strings.Contains(sql, "WHERE") {
		t.Errorf("WHERE should be present, got: %q", sql)
	}
	if strings.Contains(sql, "WHERE AND") {
		t.Errorf("leading AND should be removed, got: %q", sql)
	}
}

func TestForScalarInClause(t *testing.T) {
	dir, err := os.MkdirTemp("", "dokdo-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	writeGo(t, dir, "users.go", `
package query

type IdParams struct {
	IdList []int64
}
`)
	writeKX(t, dir, "users.kx", `
<users>
  <selectByIds set:{"users#IdParams"}>
    SELECT * FROM users
    AND id IN (
      [[ for id in idList :{
        #{id},
      }]]
    )
  </>
</>
`)

	dq := mustLoad(t, dir)

	type Params struct{ IdList []int64 }
	sql, args, err := dq.Build("users#selectByIds", Params{IdList: []int64{1, 2, 3}})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if got := strings.Count(sql, "?"); got != 3 {
		t.Errorf("'?' count: got %d, want 3 — sql: %q", got, sql)
	}
	if len(args) != 3 {
		t.Fatalf("args length: got %d, want 3", len(args))
	}
	for i, want := range []int64{1, 2, 3} {
		if args[i] != want {
			t.Errorf("args[%d]: got %v (%T), want %d", i, args[i], args[i], want)
		}
	}
}

func TestForMapUpdate(t *testing.T) {
	dir, err := os.MkdirTemp("", "dokdo-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	writeGo(t, dir, "users.go", `
package query

type UpdateParams struct {
	Id      int64
	Updates []struct {
		Key   string
		Value string
	}
}
`)
	writeKX(t, dir, "users.kx", `
<users>
  <updateUser set:{"users#UpdateParams"}>
    UPDATE users SET
    [[ for field in updates :{
      ${field.Key} = #{field.Value},
    }]]
    WHERE id = #{id}
  </>
</>
`)

	dq := mustLoad(t, dir)

	params := struct {
		Id      int64
		Updates []struct{ Key, Value string }
	}{
		Id:      1,
		Updates: []struct{ Key, Value string }{{"name", "kim"}, {"score", "90"}},
	}
	sql, args, err := dq.Build("users#updateUser", params)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if got := strings.Count(sql, "?"); got != 3 {
		t.Errorf("'?' count: got %d, want 3 — sql: %q", got, sql)
	}
	if len(args) != 3 {
		t.Fatalf("args length: got %d, want 3", len(args))
	}
	if args[2] != int64(1) {
		t.Errorf("args[2]: got %v (%T), want int64(1)", args[2], args[2])
	}
}

func TestForUnionAllNotStripped(t *testing.T) {
	dir, err := os.MkdirTemp("", "dokdo-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	writeGo(t, dir, "users.go", `
package query

type IdListParams struct {
	IdList []int64
}
`)
	writeKX(t, dir, "users.kx", `
<users>
  <selectByUnion set:{"users#IdListParams"}>
    [[ for id in idList :{
      UNION ALL SELECT * FROM users WHERE id = #{id}
    }]]
  </>
</>
`)

	dq := mustLoad(t, dir)

	type Params struct{ IdList []int64 }
	sql, _, err := dq.Build("users#selectByUnion", Params{IdList: []int64{1, 2}})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	t.Logf("SQL: %s", sql)
	if got := strings.Count(sql, "UNION ALL"); got != 2 {
		t.Errorf("UNION ALL count: got %d, want 2 — sql: %q", got, sql)
	}
}

func TestWhereWithForLoop(t *testing.T) {
	dir, err := os.MkdirTemp("", "dokdo-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	writeGo(t, dir, "users.go", `
package query

type FilterParams struct {
	Filters []struct {
		Key   string
		Value string
	}
}
`)
	writeKX(t, dir, "users.kx", `
<users>
  <selectFiltered set:{"users#FilterParams"}>
    SELECT * FROM users
    <where>
      [[ for filter in filters :{
        AND ${filter.Key} = #{filter.Value}
      }]]
    </>
  </>
</>
`)

	dq := mustLoad(t, dir)

	type Filter struct{ Key, Value string }
	type Params struct{ Filters []Filter }
	sql, args, err := dq.Build("users#selectFiltered", Params{
		Filters: []Filter{{"col1", "v1"}, {"col2", "v2"}},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	t.Logf("SQL: %s", sql)
	t.Logf("args: %v", args)
	if !strings.Contains(sql, "WHERE") {
		t.Errorf("WHERE should be present, got: %q", sql)
	}
	if strings.Contains(sql, "WHERE AND") {
		t.Errorf("leading AND should be removed by <where>, got: %q", sql)
	}
	if got := strings.Count(sql, "AND"); got != 1 {
		t.Errorf("AND count: got %d, want 1 — sql: %q", got, sql)
	}
	if got := strings.Count(sql, "?"); got != 2 {
		t.Errorf("'?' count: got %d, want 2 — sql: %q", got, sql)
	}
}

func TestIfConditionTrue(t *testing.T) {
	dir, err := os.MkdirTemp("", "dokdo-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	writeGo(t, dir, "users.go", `
package query

type GradeParams struct {
	Score int
}
`)
	writeKX(t, dir, "users.kx", `
<users>
  <selectGrade set:{"users#GradeParams"}>
    SELECT * FROM users
    [[ if score > 80 :{
      AND grade = 'A'
    }]]
  </>
</>
`)
	dq := mustLoad(t, dir)

	type Params struct{ Score int }
	sql, _, err := dq.Build("users#selectGrade", Params{Score: 90})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !strings.Contains(sql, "AND grade = 'A'") {
		t.Errorf("expected \"AND grade = 'A'\" in sql, got: %q", sql)
	}
}

func TestIfConditionFalse(t *testing.T) {
	dir, err := os.MkdirTemp("", "dokdo-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	writeGo(t, dir, "users.go", `
package query

type GradeParams struct {
	Score int
}
`)
	writeKX(t, dir, "users.kx", `
<users>
  <selectGrade set:{"users#GradeParams"}>
    SELECT * FROM users
    [[ if score > 80 :{
      AND grade = 'A'
    }]]
  </>
</>
`)
	dq := mustLoad(t, dir)

	type Params struct{ Score int }
	sql, _, err := dq.Build("users#selectGrade", Params{Score: 70})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if strings.Contains(sql, "AND grade = 'A'") {
		t.Errorf("\"AND grade = 'A'\" should be absent when score <= 80, got: %q", sql)
	}
}

func TestSwitchCaseMatch(t *testing.T) {
	dir, err := os.MkdirTemp("", "dokdo-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	writeGo(t, dir, "users.go", `
package query

type StatusParams struct {
	Status *string
}
`)
	writeKX(t, dir, "users.kx", `
<users>
  <getByStatus set:{"users#StatusParams"}>
    SELECT * FROM users
    [[ switch (status) :{
      case ("VIP") :{
        AND grade = 'A'
      }
      default :{
        AND grade = 'C'
      }
    }]]
  </>
</>
`)

	dq := mustLoad(t, dir)

	type Params struct{ Status *string }
	vip := "VIP"
	sql, _, err := dq.Build("users#getByStatus", Params{Status: &vip})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !strings.Contains(sql, "AND grade = 'A'") {
		t.Errorf("expected \"AND grade = 'A'\" for VIP, got: %q", sql)
	}
	if strings.Contains(sql, "AND grade = 'C'") {
		t.Errorf("default branch should not appear for VIP, got: %q", sql)
	}
}

func TestSubdirectory(t *testing.T) {
	dir, err := os.MkdirTemp("", "dokdo-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	writeGo(t, dir, filepath.Join("users", "detail.go"), `
package query

type DetailParams struct {
	Id int64
}
`)
	writeKX(t, dir, filepath.Join("users", "detail.kx"), `
<detail>
  <selectDetail set:{"detail#DetailParams"}>
    SELECT * FROM users WHERE id = #{id}
  </>
</>
`)

	dq := mustLoad(t, dir)

	type Params struct{ Id int64 }
	sql, args, err := dq.Build("users/detail#selectDetail", Params{Id: 1})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !strings.Contains(sql, "?") {
		t.Errorf("sql should contain '?', got: %q", sql)
	}
	if len(args) != 1 {
		t.Errorf("args length: got %d, want 1", len(args))
	}
}

// ─────────────────────────────────────────
// 에러 케이스
// ─────────────────────────────────────────

func TestBuildError_TypeNotFound(t *testing.T) {
	dir, err := os.MkdirTemp("", "dokdo-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// no .go file — "Ghost" type does not exist in dirTypes
	writeKX(t, dir, "users.kx", `
<users>
  <selectUser set:{"users#Ghost"}>
    SELECT * FROM users
  </>
</>
`)

	_, err = dokdo.Load(dir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var target *dokdo.BuildError
	if !errors.As(err, &target) {
		t.Errorf("error type: got %T (%v), want *dokdo.BuildError", err, err)
	}
}

func TestBuildError_UnsupportedType(t *testing.T) {
	dir, err := os.MkdirTemp("", "dokdo-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	writeGo(t, dir, "users.go", `
package query

type BadParams struct {
	Id   int64
	Data map[string]interface{}
}
`)
	writeKX(t, dir, "users.kx", `
<users>
  <selectUser set:{"users#BadParams"}>
    SELECT * FROM users WHERE id = #{id}
  </>
</>
`)

	_, err = dokdo.Load(dir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var target *dokdo.BuildError
	if !errors.As(err, &target) {
		t.Errorf("error type: got %T (%v), want *dokdo.BuildError", err, err)
	}
}

func TestInvalidParamsError(t *testing.T) {
	dir, err := os.MkdirTemp("", "dokdo-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	writeKX(t, dir, "users.kx", `
<users>
  <selectUser>
    SELECT * FROM users
  </>
</>
`)

	dq := mustLoad(t, dir)

	_, _, err = dq.Build("users#selectUser", map[string]interface{}{"name": "kim"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var target *dokdo.InvalidParamsError
	if !errors.As(err, &target) {
		t.Errorf("error type: got %T (%v), want *dokdo.InvalidParamsError", err, err)
	}
}

func TestTagNotFound(t *testing.T) {
	dir, err := os.MkdirTemp("", "dokdo-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	writeKX(t, dir, "users.kx", `
<users>
  <selectUser>
    SELECT * FROM users
  </>
</>
`)

	dq := mustLoad(t, dir)

	_, _, err = dq.Build("users#nonexistent", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var target *dokdo.TagNotFoundError
	if !errors.As(err, &target) {
		t.Errorf("error type: got %T (%v), want *dokdo.TagNotFoundError", err, err)
	}
}

func TestTypeMismatch(t *testing.T) {
	dir, err := os.MkdirTemp("", "dokdo-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// .go declares Name as non-pointer string; params passes *string → type mismatch
	writeGo(t, dir, "users.go", `
package query

type UserParams struct {
	Name string
}
`)
	writeKX(t, dir, "users.kx", `
<users>
  <selectUser set:{"users#UserParams"}>
    SELECT * FROM users WHERE name = #{name}
  </>
</>
`)

	dq := mustLoad(t, dir)

	type Params struct{ Name *string }
	name := "kim"
	_, _, err = dq.Build("users#selectUser", Params{Name: &name})
	if err == nil {
		t.Fatal("expected type mismatch error, got nil")
	}
	if !strings.Contains(err.Error(), "type mismatch") {
		t.Errorf("error should mention 'type mismatch', got: %v", err)
	}
}

func TestPathTraversal(t *testing.T) {
	_, err := dokdo.Load("../outside")
	if err == nil {
		t.Fatal("expected error for path traversal / non-existent path, got nil")
	}
}
