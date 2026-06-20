package builder

import (
	"strings"
	"testing"
)

func TestInjection(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string // 에러 메시지에 포함되어야 할 부분 문자열
		wantVal string // 성공 시 반환값
	}{
		// ── 결함 A: 하이픈 우회 ──────────────────────────────────────────────
		// rawPattern은 '-'을 허용하지만, DROP-X 전체가 blocklist에 없어 통과됨
		{"A/DROP-x", "DROP-x", "blocked SQL keyword", ""},
		{"A/DELETE-rows", "DELETE-rows", "blocked SQL keyword", ""},
		{"A/UPDATE-all", "UPDATE-all", "blocked SQL keyword", ""},
		{"A/TRUNCATE-t", "TRUNCATE-t", "blocked SQL keyword", ""},
		{"A/SELECT-col", "SELECT-col", "blocked SQL keyword", ""},
		{"A/CREATE-tmp", "CREATE-tmp", "blocked SQL keyword", ""},
		{"A/ALTER-t", "ALTER-t", "blocked SQL keyword", ""},

		// ── 결함 B: 백틱 페어 검증 우회 ─────────────────────────────────────
		// `` — btCount==2이지만 len==2 < 3, 내부가 비어 있음
		{"B/empty-backtick-pair", "``", "empty backtick identifier", ""},
		// ` — 단일 백틱, btCount==1
		{"B/single-backtick", "`", "invalid backtick usage", ""},
		// ```x``` — btCount==6
		{"B/triple-backtick", "```x```", "invalid backtick usage", ""},
		// `a`b` — btCount==3 (앞뒤+내부)
		{"B/three-backtick", "`a`b`", "invalid backtick usage", ""},
		// `x``y` — btCount==4
		{"B/four-backtick", "`x``y`", "invalid backtick usage", ""},

		// ── 결함 C: 도트 우회 ────────────────────────────────────────────────
		// ID.DROP 전체가 blocklist에 없어 통과됨
		{"C/id.DROP", "id.DROP", "blocked SQL keyword", ""},
		{"C/table.SELECT", "table.SELECT", "blocked SQL keyword", ""},
		{"C/col.DELETE", "col.DELETE", "blocked SQL keyword", ""},
		{"C/t.TRUNCATE", "t.TRUNCATE", "blocked SQL keyword", ""},
		{"C/a.UPDATE", "a.UPDATE", "blocked SQL keyword", ""},

		// ── 결함 D: 유니코드 우회 ────────────────────────────────────────────
		{"D/umlaut-plain", "üser", "non-ASCII character in identifier", ""},
		{"D/umlaut-table", "täble", "non-ASCII character in identifier", ""},
		// 백틱 내부 비ASCII
		{"D/umlaut-backtick", "`über`", "non-ASCII character in identifier", ""},

		// ── 결함 F: 연속 하이픈 (SQL 주석 주입) ─────────────────────────────
		{"F/SE--LECT", "SE--LECT", "consecutive hyphens not allowed", ""},
		{"F/x--", "x--", "consecutive hyphens not allowed", ""},
		{"F/--x", "--x", "invalid identifier", ""},
		{"F/a--b", "a--b", "consecutive hyphens not allowed", ""},

		// ── 결함 G: 숫자로 시작하는 식별자 ──────────────────────────────────
		{"G/digit-only", "1", "invalid identifier", ""},
		{"G/float-like", "1.5", "invalid identifier", ""},
		{"G/digit-alpha", "123abc", "invalid identifier", ""},

		// ── 결함 E: 길이 제한 없음 ───────────────────────────────────────────
		{"E/len-129", strings.Repeat("a", 129), "identifier too long", ""},
		{"E/len-1000", strings.Repeat("b", 1000), "identifier too long", ""},
		{"E/len-10000", strings.Repeat("c", 10000), "identifier too long", ""},

		// ── 회귀: 통과해야 하는 정상 케이스 ─────────────────────────────────
		{"R/users", "users", "", "users"},
		{"R/user_id", "user_id", "", "user_id"},
		{"R/t.column", "t.column", "", "t.column"},
		{"R/order_id", "order_id", "", "order_id"},
		{"R/created_at", "created_at", "", "created_at"},
		{"R/total_amount", "total_amount", "", "total_amount"},
		{"R/o.status", "o.status", "", "o.status"},
		{"R/ASC", "ASC", "", "ASC"},
		{"R/DESC", "DESC", "", "DESC"},
		// 백틱 allowlist 키워드 — 통과해야 함
		{"R/backtick-IN", "`IN`", "", "`IN`"},
		{"R/backtick-STATUS", "`STATUS`", "", "`STATUS`"},
		{"R/backtick-myview", "`myview`", "", "`myview`"},
		// 정확히 128자 — 경계값, 통과해야 함
		{"R/len-128", strings.Repeat("a", 128), "", strings.Repeat("a", 128)},

		// ── 회귀: 차단되어야 하는 케이스 ─────────────────────────────────────
		{"RB/SELECT", "SELECT", "blocked SQL keyword", ""},
		{"RB/drop", "drop", "blocked SQL keyword", ""},
		{"RB/DELETE", "DELETE", "blocked SQL keyword", ""},
		{"RB/TRUNCATE", "TRUNCATE", "blocked SQL keyword", ""},
		{"RB/IN-no-backtick", "IN", "reserved word requires backtick", ""},
		{"RB/COUNT-no-backtick", "COUNT", "reserved word requires backtick", ""},
		{"RB/STATUS-no-backtick", "STATUS", "reserved word requires backtick", ""},
		// 백틱 안에 blocklist 키워드 — 차단되어야 함
		{"RB/backtick-SELECT", "`SELECT`", "blocked SQL keyword", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateRaw(tt.input)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("validateRaw(%q): 에러 %q 기대했으나 nil 반환 (got %q)", tt.input, tt.wantErr, got)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("validateRaw(%q): 에러 %q에 %q 포함되지 않음", tt.input, err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("validateRaw(%q): 예상치 못한 에러: %v", tt.input, err)
			}
			if got != tt.wantVal {
				t.Errorf("validateRaw(%q) = %q, want %q", tt.input, got, tt.wantVal)
			}
		})
	}
}
