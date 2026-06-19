package builder

import (
	"strings"
	"testing"
)

// ─── validateRaw ────────────────────────────────────────────────────────────

func TestValidateRaw(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr string
	}{
		{"일반 식별자", "users", "users", ""},
		{"언더스코어", "user_id", "user_id", ""},
		{"점 표기", "t.column", "t.column", ""},
		{"숫자 포함", "col1", "col1", ""},
		{"백틱 allowlist", "`IN`", "`IN`", ""},
		{"백틱 임의 식별자", "`myview`", "`myview`", ""},
		{"blocklist 대문자", "SELECT", "", "blocked SQL keyword"},
		{"blocklist 소문자", "drop", "", "blocked SQL keyword"},
		{"allowlist 백틱 없음 IN", "IN", "", "reserved word requires backtick"},
		{"allowlist 백틱 없음 COUNT", "COUNT", "", "reserved word requires backtick"},
		{"정규식 위반 공백", "user name", "", "invalid identifier"},
		{"정규식 위반 세미콜론", "user;drop", "", "invalid identifier"},
		{"백틱 blocklist", "`SELECT`", "", "blocked SQL keyword"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateRaw(tt.input)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("validateRaw(%q): want error containing %q, got nil", tt.input, tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("validateRaw(%q): error %q does not contain %q", tt.input, err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("validateRaw(%q): unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("validateRaw(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ─── evalCond ───────────────────────────────────────────────────────────────

func TestEvalCond(t *testing.T) {
	nilPtr := (*int)(nil)
	nonNilPtr := new(int)

	tests := []struct {
		name    string
		cond    string
		params  interface{}
		want    bool
		wantErr bool
	}{
		{"> true", "age > 10", map[string]interface{}{"age": 20}, true, false},
		{"< false", "age < 10", map[string]interface{}{"age": 20}, false, false},
		{">= equal", "age >= 20", map[string]interface{}{"age": 20}, true, false},
		{"<= less", "age <= 19", map[string]interface{}{"age": 20}, false, false},
		{"== num true", "age == 20", map[string]interface{}{"age": 20}, true, false},
		{"<> num false", "age <> 20", map[string]interface{}{"age": 20}, false, false},
		{"== nil: nil ptr", "ptr == nil", map[string]interface{}{"ptr": nilPtr}, true, false},
		{"<> nil: nil ptr", "ptr <> nil", map[string]interface{}{"ptr": nilPtr}, false, false},
		{"== nil: non-nil ptr", "ptr == nil", map[string]interface{}{"ptr": nonNilPtr}, false, false},
		{"nil 잘못된 연산자", "ptr > nil", map[string]interface{}{"ptr": nilPtr}, false, true},
		{"문자열 == true", "name == alice", map[string]interface{}{"name": "alice"}, true, false},
		{"문자열 <> true", "name <> alice", map[string]interface{}{"name": "bob"}, true, false},
		{"문자열 > 에러", "name > alice", map[string]interface{}{"name": "alice"}, false, true},
		{"잘못된 연산자", "age # 10", map[string]interface{}{"age": 5}, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := evalCond(tt.cond, tt.params)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("evalCond(%q): want error, got nil", tt.cond)
				}
				return
			}
			if err != nil {
				t.Fatalf("evalCond(%q): unexpected error: %v", tt.cond, err)
			}
			if got != tt.want {
				t.Errorf("evalCond(%q) = %v, want %v", tt.cond, got, tt.want)
			}
		})
	}
}

// ─── resolveValue ───────────────────────────────────────────────────────────

type testAddr struct{ City string }
type testPerson struct {
	Name string
	Addr testAddr
}

func TestResolveValue(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		params  interface{}
		want    interface{}
		wantErr string
	}{
		{
			"struct 필드",
			"Name",
			testPerson{Name: "Alice"},
			"Alice",
			"",
		},
		{
			"map 키",
			"name",
			map[string]interface{}{"name": "Alice"},
			"Alice",
			"",
		},
		{
			"점 표기 중첩 struct",
			"Addr.City",
			testPerson{Addr: testAddr{City: "Seoul"}},
			"Seoul",
			"",
		},
		{
			"포인터 역참조",
			"Name",
			&testPerson{Name: "Bob"},
			"Bob",
			"",
		},
		{
			"nil 포인터",
			"Name",
			(*testPerson)(nil),
			nil,
			"",
		},
		{
			"struct 필드 없음",
			"Missing",
			testPerson{},
			nil,
			"field",
		},
		{
			"map 키 없음",
			"missing",
			map[string]interface{}{"name": "Alice"},
			nil,
			"key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveValue(tt.path, tt.params)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("resolveValue(%q): want error containing %q, got nil", tt.path, tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("resolveValue(%q): error %q does not contain %q", tt.path, err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveValue(%q): unexpected error: %v", tt.path, err)
			}
			if got != tt.want {
				t.Errorf("resolveValue(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

// ─── trimLeadingAndOr ───────────────────────────────────────────────────────

func TestTrimLeadingAndOr(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"AND 제거", "AND col1 = 1", "col1 = 1"},
		{"OR 제거", "OR col1 = 1", "col1 = 1"},
		{"and 소문자", "and col1 = 1", "col1 = 1"},
		{"or 소문자", "or col1 = 1", "col1 = 1"},
		{"해당 없음", "col1 = 1", "col1 = 1"},
		{"빈 문자열", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trimLeadingAndOr(tt.input)
			if got != tt.want {
				t.Errorf("trimLeadingAndOr(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ─── removeTrailingComma ────────────────────────────────────────────────────

func TestRemoveTrailingComma(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"trailing comma 제거", "a, b, c,", "a, b, c"},
		{"comma 없음", "a, b, c", "a, b, c"},
		{"공백 + trailing comma", "  value,  ", "value"},
		{"빈 문자열", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := removeTrailingComma(tt.input)
			if got != tt.want {
				t.Errorf("removeTrailingComma(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
