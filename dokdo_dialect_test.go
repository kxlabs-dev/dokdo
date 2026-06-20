package dokdo_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/kxlabs-dev/dokdo"
)

func mustLoadDialect(t *testing.T, dir string, d dokdo.Dialect) *dokdo.Dokdo {
	t.Helper()
	dq, err := dokdo.Load(dir, d)
	if err != nil {
		t.Fatalf("Load(%q, dialect=%v): %v", dir, d, err)
	}
	return dq
}

// verifyPlaceholders checks that:
//  1. The SQL contains exactly wantN placeholders in the expected dialect format
//  2. The args slice has exactly wantN entries
func verifyPlaceholders(t *testing.T, sql string, args []interface{}, wantN int, d dokdo.Dialect) {
	t.Helper()
	switch d {
	case dokdo.DialectMySQL:
		got := strings.Count(sql, "?")
		if got != wantN {
			t.Errorf("MySQL ? count: got %d, want %d — sql: %q", got, wantN, sql)
		}
	case dokdo.DialectPostgres:
		for i := 1; i <= wantN; i++ {
			ph := fmt.Sprintf("$%d", i)
			if !strings.Contains(sql, ph) {
				t.Errorf("Postgres missing %s — sql: %q", ph, sql)
			}
		}
		if strings.Contains(sql, fmt.Sprintf("$%d", wantN+1)) {
			t.Errorf("Postgres unexpected $%d — sql: %q", wantN+1, sql)
		}
	case dokdo.DialectOracle:
		for i := 1; i <= wantN; i++ {
			ph := fmt.Sprintf(":%d", i)
			if !strings.Contains(sql, ph) {
				t.Errorf("Oracle missing %s — sql: %q", ph, sql)
			}
		}
		if strings.Contains(sql, fmt.Sprintf(":%d", wantN+1)) {
			t.Errorf("Oracle unexpected :%d — sql: %q", wantN+1, sql)
		}
	case dokdo.DialectSQLServer:
		for i := 1; i <= wantN; i++ {
			ph := fmt.Sprintf("@p%d", i)
			if !strings.Contains(sql, ph) {
				t.Errorf("SQLServer missing %s — sql: %q", ph, sql)
			}
		}
		if strings.Contains(sql, fmt.Sprintf("@p%d", wantN+1)) {
			t.Errorf("SQLServer unexpected @p%d — sql: %q", wantN+1, sql)
		}
	}
	if len(args) != wantN {
		t.Errorf("args length: got %d, want %d", len(args), wantN)
	}
}

// TestDialect_DefaultIsMySQL confirms Load without dialect arg produces ? (backward compat).
func TestDialect_DefaultIsMySQL(t *testing.T) {
	dir := t.TempDir()
	writeGo(t, dir, "orders.go", ordersGo)
	writeKX(t, dir, "orders.kx", ordersKX)
	dq := mustLoad(t, dir) // dialect 생략 → MySQL 기본값

	cancelled := "cancelled"
	params := struct {
		Ids    []int64
		Status *string
	}{Ids: []int64{1, 2, 3}, Status: &cancelled}

	sql, args, err := dq.Build("orders#deleteOrders", params)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	t.Logf("SQL: %s", sql)
	t.Logf("args: %v", args)
	verifyPlaceholders(t, sql, args, 4, dokdo.DialectMySQL)
}

// TestDialect_AllFour runs representative queries through all 4 dialects
// and verifies placeholder format, numbering continuity, and arg order.
func TestDialect_AllFour(t *testing.T) {
	dialects := []struct {
		name    string
		dialect dokdo.Dialect
	}{
		{"MySQL", dokdo.DialectMySQL},
		{"Postgres", dokdo.DialectPostgres},
		{"Oracle", dokdo.DialectOracle},
		{"SQLServer", dokdo.DialectSQLServer},
	}

	for _, dt := range dialects {
		dt := dt
		t.Run(dt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeGo(t, dir, "orders.go", ordersGo)
			writeKX(t, dir, "orders.kx", ordersKX)
			dq := mustLoadDialect(t, dir, dt.dialect)

			// ── Case A: deleteOrders
			// WHERE( FOR(#{id}×3) AND #{status} )
			// 플레이스홀더 순서: id1, id2, id3, status → 4개
			t.Run("deleteOrders_WHERE_FOR", func(t *testing.T) {
				cancelled := "cancelled"
				params := struct {
					Ids    []int64
					Status *string
				}{Ids: []int64{10, 20, 30}, Status: &cancelled}

				sql, args, err := dq.Build("orders#deleteOrders", params)
				if err != nil {
					t.Fatalf("Build: %v", err)
				}
				t.Logf("[%s] SQL: %s", dt.name, sql)
				t.Logf("[%s] args: %v", dt.name, args)

				verifyPlaceholders(t, sql, args, 4, dt.dialect)

				if args[0] != int64(10) {
					t.Errorf("args[0]: got %v, want 10", args[0])
				}
				if args[1] != int64(20) {
					t.Errorf("args[1]: got %v, want 20", args[1])
				}
				if args[2] != int64(30) {
					t.Errorf("args[2]: got %v, want 30", args[2])
				}
				if s, ok := args[3].(*string); !ok || *s != "cancelled" {
					t.Errorf("args[3]: got %v (%T), want *string(cancelled)", args[3], args[3])
				}
			})

			// ── Case B: updateOrderStatus
			// FOR( ${key}=#{value}×2 ) WHERE id=#{id}
			// 플레이스홀더 순서: value1, value2, id → 3개
			t.Run("updateOrderStatus_FOR_then_bind", func(t *testing.T) {
				params := struct {
					Id      int64
					Updates []struct{ Key, Value string }
				}{
					Id:      int64(99),
					Updates: []struct{ Key, Value string }{{"name", "shipped"}, {"grade", "S"}},
				}

				sql, args, err := dq.Build("orders#updateOrderStatus", params)
				if err != nil {
					t.Fatalf("Build: %v", err)
				}
				t.Logf("[%s] SQL: %s", dt.name, sql)
				t.Logf("[%s] args: %v", dt.name, args)

				verifyPlaceholders(t, sql, args, 3, dt.dialect)

				if args[0] != "shipped" {
					t.Errorf("args[0]: got %v, want shipped", args[0])
				}
				if args[1] != "S" {
					t.Errorf("args[1]: got %v, want S", args[1])
				}
				if args[2] != int64(99) {
					t.Errorf("args[2]: got %v, want 99", args[2])
				}
			})

			// ── Case C: selectPaged
			// WHERE(#{keyword} #{status}) LIMIT #{limit} OFFSET #{offset}
			// 플레이스홀더 순서: keyword, status, limit, offset → 4개
			// WHERE 절 안 바인드 + 절 밖 바인드가 번호가 이어지는지 확인
			t.Run("selectPaged_WHERE_then_limit_offset", func(t *testing.T) {
				kw := "%kim%"
				active := "active"
				params := struct {
					Keyword *string
					Status  *string
					Limit   int64
					Offset  int64
				}{Keyword: &kw, Status: &active, Limit: int64(10), Offset: int64(20)}

				sql, args, err := dq.Build("orders#selectPaged", params)
				if err != nil {
					t.Fatalf("Build: %v", err)
				}
				t.Logf("[%s] SQL: %s", dt.name, sql)
				t.Logf("[%s] args: %v", dt.name, args)

				verifyPlaceholders(t, sql, args, 4, dt.dialect)

				if args[2] != int64(10) {
					t.Errorf("args[2]: got %v, want int64(10)", args[2])
				}
				if args[3] != int64(20) {
					t.Errorf("args[3]: got %v, want int64(20)", args[3])
				}
			})

			// ── Case D: bulkInsertOrders
			// FOR( (#{Name},#{Amount}) × 3 )
			// 플레이스홀더 순서: name1,amt1, name2,amt2, name3,amt3 → 6개
			t.Run("bulkInsert_FOR_only", func(t *testing.T) {
				params := struct {
					Orders []struct {
						Name   string
						Amount int64
					}
				}{
					Orders: []struct {
						Name   string
						Amount int64
					}{
						{"A", int64(100)},
						{"B", int64(200)},
						{"C", int64(300)},
					},
				}

				sql, args, err := dq.Build("orders#bulkInsertOrders", params)
				if err != nil {
					t.Fatalf("Build: %v", err)
				}
				t.Logf("[%s] SQL: %s", dt.name, sql)
				t.Logf("[%s] args: %v", dt.name, args)

				verifyPlaceholders(t, sql, args, 6, dt.dialect)

				if args[0] != "A" {
					t.Errorf("args[0]: got %v, want A", args[0])
				}
				if args[1] != int64(100) {
					t.Errorf("args[1]: got %v, want 100", args[1])
				}
				if args[4] != "C" {
					t.Errorf("args[4]: got %v, want C", args[4])
				}
				if args[5] != int64(300) {
					t.Errorf("args[5]: got %v, want 300", args[5])
				}
			})
		})
	}
}

// TestDialect_LastDialectWins confirms that passing multiple dialect args uses the last one.
func TestDialect_LastDialectWins(t *testing.T) {
	dir := t.TempDir()
	writeGo(t, dir, "orders.go", ordersGo)
	writeKX(t, dir, "orders.kx", ordersKX)

	dq, err := dokdo.Load(dir, dokdo.DialectMySQL, dokdo.DialectPostgres)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// selectPaged는 keyword/status 필드를 조건 평가 시 resolveValue로 참조하므로
	// 필드가 없으면 RuntimeError 발생. nil 포인터로 포함해야 한다.
	params := struct {
		Keyword *string
		Status  *string
		Limit   int64
		Offset  int64
	}{Keyword: nil, Status: nil, Limit: 5, Offset: 0}

	sql, args, err := dq.Build("orders#selectPaged", params)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	t.Logf("SQL: %s", sql)
	t.Logf("args: %v", args)

	if !strings.Contains(sql, "$1") {
		t.Errorf("expected $1 (Postgres is last dialect) — sql: %q", sql)
	}
	if strings.Contains(sql, "?") {
		t.Errorf("? should be absent (not MySQL) — sql: %q", sql)
	}
	_ = args
}
