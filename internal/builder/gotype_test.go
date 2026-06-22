package builder

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempGo(t *testing.T, dir, relPath, content string) string {
	t.Helper()
	full := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile %s: %v", relPath, err)
	}
	return full
}

func TestParseGoFile(t *testing.T) {
	tests := []struct {
		name      string
		relPath   string
		content   string
		wantTypes map[string][]string // typeName → field names
		wantErr   bool
	}{
		{
			// 회귀: 같은 디렉토리의 users.go → UserParams 정상 반환
			name:    "regression/same-dir",
			relPath: "users.go",
			content: `package query
type UserParams struct {
	Name   *string
	Score  int
}
`,
			wantTypes: map[string][]string{
				"UserParams": {"Name", "Score"},
			},
		},
		{
			// 신규: 서브디렉토리 common/shared.go → SharedParams 정상 반환
			name:    "cross-dir/subdir",
			relPath: "common/shared.go",
			content: `package common
type SharedParams struct {
	Id     int64
	Status *string
}
`,
			wantTypes: map[string][]string{
				"SharedParams": {"Id", "Status"},
			},
		},
		{
			// unexported 타입은 맵에 포함되지 않아야 함
			name:    "unexported-type-filtered",
			relPath: "types.go",
			content: `package query
type PublicParams struct {
	Name string
}
type privateParams struct {
	secret string
}
`,
			wantTypes: map[string][]string{
				"PublicParams": {"Name"},
			},
		},
		{
			// 파일 없음 → error 반환
			name:    "file-not-found",
			relPath: "nonexistent.go",
			wantErr: true,
		},
		{
			// []CustomType → error 반환
			name:    "unsupported-custom-slice",
			relPath: "bad.go",
			content: `package query
type CustomField struct{ Val string }
type BadParams struct {
	Items []CustomField
}
`,
			wantErr: true,
		},
		{
			// map 필드 → error 반환
			name:    "unsupported-map",
			relPath: "bad2.go",
			content: `package query
type BadParams struct {
	Data map[string]interface{}
}
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			var filePath string
			if tt.content != "" {
				filePath = writeTempGo(t, dir, tt.relPath, tt.content)
			} else {
				filePath = filepath.Join(dir, tt.relPath)
			}

			result, err := ParseGoFile(filePath)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseGoFile(%q): expected error, got nil", tt.relPath)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseGoFile(%q): unexpected error: %v", tt.relPath, err)
			}

			for typeName, wantFields := range tt.wantTypes {
				info, ok := result[typeName]
				if !ok {
					t.Errorf("type %q not found in result (got keys: %v)", typeName, mapKeys(result))
					continue
				}
				gotFields := fieldNames(info)
				if len(gotFields) != len(wantFields) {
					t.Errorf("type %q: field count got %d, want %d (got %v, want %v)",
						typeName, len(gotFields), len(wantFields), gotFields, wantFields)
					continue
				}
				for i, wf := range wantFields {
					if gotFields[i] != wf {
						t.Errorf("type %q field[%d]: got %q, want %q", typeName, i, gotFields[i], wf)
					}
				}
			}
			// unexported-type-filtered: private type must not appear
			if _, ok := result["privateParams"]; ok {
				t.Error("unexported type 'privateParams' should not appear in result")
			}
		})
	}
}

func TestParseGoFile_SameTypeNameDifferentFiles(t *testing.T) {
	// a/types.go와 b/types.go가 둘 다 Params라는 타입을 선언할 때
	// 각 ParseGoFile 호출은 독립된 맵을 반환하고 서로 충돌하지 않아야 한다.
	dir := t.TempDir()
	aFile := writeTempGo(t, dir, "a/types.go", `package a
type Params struct {
	Alpha string
}
`)
	bFile := writeTempGo(t, dir, "b/types.go", `package b
type Params struct {
	Beta int64
}
`)

	aResult, err := ParseGoFile(aFile)
	if err != nil {
		t.Fatalf("ParseGoFile(a/types.go): %v", err)
	}
	bResult, err := ParseGoFile(bFile)
	if err != nil {
		t.Fatalf("ParseGoFile(b/types.go): %v", err)
	}

	aInfo, ok := aResult["Params"]
	if !ok {
		t.Fatal("Params not found in a/types.go result")
	}
	bInfo, ok := bResult["Params"]
	if !ok {
		t.Fatal("Params not found in b/types.go result")
	}

	if len(aInfo.Fields) != 1 || aInfo.Fields[0].Name != "Alpha" {
		t.Errorf("a/Params fields: got %v, want [Alpha]", fieldNames(aInfo))
	}
	if len(bInfo.Fields) != 1 || bInfo.Fields[0].Name != "Beta" {
		t.Errorf("b/Params fields: got %v, want [Beta]", fieldNames(bInfo))
	}
}

func mapKeys(m map[string]*TypeInfo) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func fieldNames(info *TypeInfo) []string {
	names := make([]string, len(info.Fields))
	for i, f := range info.Fields {
		names[i] = f.Name
	}
	return names
}
