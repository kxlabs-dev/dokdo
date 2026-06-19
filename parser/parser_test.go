package parser

import (
	"strings"
	"testing"
)

// findNode returns the first node of type T from nodes, searching recursively.
func findNode[T SQLNode](nodes []SQLNode) (T, bool) {
	var zero T
	for _, n := range nodes {
		if v, ok := n.(T); ok {
			return v, true
		}
	}
	return zero, false
}

// containsNode checks whether any node in the slice is of type T.
func containsNode[T SQLNode](nodes []SQLNode) bool {
	_, ok := findNode[T](nodes)
	return ok
}

func TestParseFile(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		input := `
<users>
  <selectUser set:{"users#UserParams"}>
    SELECT * FROM users
  </>
</>`
		qf, err := ParseFile(input, "users.kx")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if qf.Namespace != "users" {
			t.Errorf("namespace: got %q, want %q", qf.Namespace, "users")
		}
		if len(qf.Queries) != 1 {
			t.Fatalf("queries: got %d, want 1", len(qf.Queries))
		}
		q := qf.Queries[0]
		if q.Name != "selectUser" {
			t.Errorf("query name: got %q, want %q", q.Name, "selectUser")
		}
		if q.ParamRef != "users#UserParams" {
			t.Errorf("param ref: got %q, want %q", q.ParamRef, "users#UserParams")
		}
	})

	t.Run("if_else_if_else", func(t *testing.T) {
		input := `
<users>
  <selectUser set:{"users#UserParams"}>
    SELECT * FROM users
    [[ if score > 90 :{
      AND grade = 'A'
    } else if score > 80 :{
      AND grade = 'B'
    } else :{
      AND grade = 'C'
    }]]
  </>
</>`
		qf, err := ParseFile(input, "users.kx")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		q := qf.Queries[0]
		ifn, ok := findNode[*IfNode](q.Body)
		if !ok {
			t.Fatal("IfNode not found in body")
		}
		if ifn.Cond != "score > 90" {
			t.Errorf("if cond: got %q, want %q", ifn.Cond, "score > 90")
		}
		if len(ifn.ElseIfs) != 1 {
			t.Fatalf("else-if count: got %d, want 1", len(ifn.ElseIfs))
		}
		if ifn.ElseIfs[0].Cond != "score > 80" {
			t.Errorf("else-if cond: got %q, want %q", ifn.ElseIfs[0].Cond, "score > 80")
		}
		if ifn.Else == nil {
			t.Error("else branch is nil")
		}
	})

	t.Run("if_not_equal", func(t *testing.T) {
		input := `
<users>
  <selectUser set:{"users#UserParams"}>
    [[ if name != nil :{
      AND name = #{name}
    } else if score != nil :{
      AND score = #{score}
    }]]
  </>
</>`
		qf, err := ParseFile(input, "users.kx")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		ifn, ok := findNode[*IfNode](qf.Queries[0].Body)
		if !ok {
			t.Fatal("IfNode not found in body")
		}
		if ifn.Cond != "name <> nil" {
			t.Errorf("if cond: got %q, want %q", ifn.Cond, "name <> nil")
		}
		if len(ifn.ElseIfs) != 1 {
			t.Fatalf("else-if count: got %d, want 1", len(ifn.ElseIfs))
		}
		if ifn.ElseIfs[0].Cond != "score <> nil" {
			t.Errorf("else-if cond: got %q, want %q", ifn.ElseIfs[0].Cond, "score <> nil")
		}
	})

	t.Run("switch_case_default", func(t *testing.T) {
		input := `
<users>
  <getByStatus set:{"users#StatusParams"}>
    [[ switch (status) :{
      case ("VIP") :{
        AND grade = 'A'
      }
      case ("BASIC") :{
        AND grade = 'B'
      }
      default :{
        AND grade = 'C'
      }
    }]]
  </>
</>`
		qf, err := ParseFile(input, "users.kx")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		q := qf.Queries[0]
		sw, ok := findNode[*SwitchNode](q.Body)
		if !ok {
			t.Fatal("SwitchNode not found in body")
		}
		if sw.Expr != "status" {
			t.Errorf("switch expr: got %q, want %q", sw.Expr, "status")
		}
		if len(sw.Cases) != 2 {
			t.Fatalf("case count: got %d, want 2", len(sw.Cases))
		}
		if sw.Cases[0].Value != "VIP" {
			t.Errorf("case[0] value: got %q, want %q", sw.Cases[0].Value, "VIP")
		}
		if sw.Cases[1].Value != "BASIC" {
			t.Errorf("case[1] value: got %q, want %q", sw.Cases[1].Value, "BASIC")
		}
		if sw.Default == nil {
			t.Error("default branch is nil")
		}
	})

	t.Run("for_scalar", func(t *testing.T) {
		input := `
<users>
  <selectByIds set:{"users#IdParams"}>
    AND id IN (
      [[ for id in idList :{
        #{id},
      }]]
    )
  </>
</>`
		qf, err := ParseFile(input, "users.kx")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		fn, ok := findNode[*ForNode](qf.Queries[0].Body)
		if !ok {
			t.Fatal("ForNode not found")
		}
		if fn.ItemVar != "id" {
			t.Errorf("ItemVar: got %q, want %q", fn.ItemVar, "id")
		}
		if fn.Collection != "idList" {
			t.Errorf("Collection: got %q, want %q", fn.Collection, "idList")
		}
		if !containsNode[*BindParam](fn.Body) {
			t.Error("expected BindParam in for body")
		}
		bp, _ := findNode[*BindParam](fn.Body)
		if bp.Path != "id" {
			t.Errorf("bind path: got %q, want %q", bp.Path, "id")
		}
	})

	t.Run("for_struct", func(t *testing.T) {
		input := `
<users>
  <selectCases set:{"users#CaseParams"}>
    CASE
      [[ for item in cases :{
        WHEN ${item.Grade} = #{item.Score} THEN #{item.Result}
      }]]
    END
  </>
</>`
		qf, err := ParseFile(input, "users.kx")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		fn, ok := findNode[*ForNode](qf.Queries[0].Body)
		if !ok {
			t.Fatal("ForNode not found")
		}
		if fn.ItemVar != "item" {
			t.Errorf("ItemVar: got %q, want %q", fn.ItemVar, "item")
		}
		if fn.Collection != "cases" {
			t.Errorf("Collection: got %q, want %q", fn.Collection, "cases")
		}
		rp, ok := findNode[*RawParam](fn.Body)
		if !ok {
			t.Fatal("expected RawParam in for body")
		}
		if rp.Path != "item.Grade" {
			t.Errorf("raw path: got %q, want %q", rp.Path, "item.Grade")
		}
		bp, ok := findNode[*BindParam](fn.Body)
		if !ok {
			t.Fatal("expected BindParam in for body")
		}
		if bp.Path != "item.Score" {
			t.Errorf("bind path: got %q, want %q", bp.Path, "item.Score")
		}
	})

	t.Run("where_with_if", func(t *testing.T) {
		input := `
<users>
  <selectUser set:{"users#UserParams"}>
    SELECT * FROM users
    <where>
      [[ if score > 0 :{
        AND score = #{score}
      }]]
    </>
  </>
</>`
		qf, err := ParseFile(input, "users.kx")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		wn, ok := findNode[*WhereNode](qf.Queries[0].Body)
		if !ok {
			t.Fatal("WhereNode not found")
		}
		if !containsNode[*IfNode](wn.Children) {
			t.Error("expected IfNode inside WhereNode")
		}
	})

	t.Run("nested_if_in_for", func(t *testing.T) {
		input := `
<users>
  <selectComplex set:{"users#UserParams"}>
    [[ if count > 0 :{
      AND order_id IN (
        [[ for id in orders :{
          #{id},
        }]]
      )
    }]]
  </>
</>`
		qf, err := ParseFile(input, "users.kx")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		ifn, ok := findNode[*IfNode](qf.Queries[0].Body)
		if !ok {
			t.Fatal("IfNode not found")
		}
		if ifn.Cond != "count > 0" {
			t.Errorf("if cond: got %q, want %q", ifn.Cond, "count > 0")
		}
		if !containsNode[*ForNode](ifn.Then) {
			t.Error("expected ForNode inside IfNode.Then")
		}
		fn, _ := findNode[*ForNode](ifn.Then)
		if fn.Collection != "orders" {
			t.Errorf("for collection: got %q, want %q", fn.Collection, "orders")
		}
	})

	t.Run("comment_skip", func(t *testing.T) {
		input := `
<!-- file header comment -->
<users>
  <!-- section comment -->
  <selectUser set:{"users#UserParams"}>
    <!-- inline comment --> SELECT * FROM users
  </>
</>`
		qf, err := ParseFile(input, "users.kx")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if qf.Namespace != "users" {
			t.Errorf("namespace: got %q", qf.Namespace)
		}
		if len(qf.Queries) != 1 {
			t.Errorf("queries: got %d, want 1", len(qf.Queries))
		}
	})

	t.Run("escapes", func(t *testing.T) {
		input := `
<users>
  <selectUser>
    AND a != b
    AND c \> d
    AND e \< f
    AND g \>= h
    AND i \<= j
  </>
</>`
		qf, err := ParseFile(input, "users.kx")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		body := qf.Queries[0].Body
		collect := func(nodes []SQLNode) []string {
			var out []string
			for _, n := range nodes {
				if st, ok := n.(*SQLText); ok {
					out = append(out, st.Text)
				}
			}
			return out
		}
		texts := collect(body)
		joined := strings.Join(texts, "")
		for _, want := range []string{"<>", ">", "<", ">=", "<="} {
			if !strings.Contains(joined, want) {
				t.Errorf("expected %q in body text, got: %q", want, joined)
			}
		}
	})
}

func TestParseFileErrors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		file    string
		wantMsg string
	}{
		{
			name: "root_tag_mismatch",
			input: `<users>
  <selectUser>
    SELECT * FROM users
  </>
</>`,
			file:    "orders.kx",
			wantMsg: "does not match filename",
		},
		{
			name: "sql_keyword_select",
			input: `<users>
  <select>
    SELECT * FROM users
  </>
</>`,
			file:    "users.kx",
			wantMsg: "reserved SQL keyword",
		},
		{
			name: "sql_keyword_from",
			input: `<users>
  <from>
    SELECT * FROM users
  </>
</>`,
			file:    "users.kx",
			wantMsg: "reserved SQL keyword",
		},
		{
			name: "raw_outside_for",
			input: `<users>
  <selectUser>
    SELECT ${col} FROM users
  </>
</>`,
			file:    "users.kx",
			wantMsg: "only allowed inside a 'for'",
		},
		{
			name: "for_map_key_wildcard",
			input: `<users>
  <selectDynamic>
    [[ for key, value in filters :{
      AND #{value}
    }]]
  </>
</>`,
			file:    "users.kx",
			wantMsg: "removed in v2.1",
		},
		{
			name: "empty_bind",
			input: `<users>
  <selectUser>
    AND name = #{}
  </>
</>`,
			file:    "users.kx",
			wantMsg: "",
		},
		{
			name: "empty_raw",
			input: `<users>
  <selectUser>
    [[ for id in ids :{
      ${}
    }]]
  </>
</>`,
			file:    "users.kx",
			wantMsg: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseFile(tc.input, tc.file)
			if err == nil {
				t.Errorf("expected error, got nil")
				return
			}
			if tc.wantMsg != "" && !strings.Contains(err.Error(), tc.wantMsg) {
				t.Errorf("error message:\n  got:  %q\n  want substring: %q", err.Error(), tc.wantMsg)
			}
		})
	}
}
