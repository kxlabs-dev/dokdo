package dokdo_test

import (
	"strings"
	"testing"

	"github.com/kxlabs-dev/dokdo"
)

const ordersGo = `
package query

// Q1
type SearchParams struct {
	Status    *string
	Ids       []int64
	Grade     *string
	Keyword   *string
	StartDate *string
	Filters   []struct {
		Key   string
		Value string
	}
}

// Q2
type AdvancedParams struct {
	MinScore  *int
	MaxScore  *int
	MinAmount *int64
	MaxAmount *int64
}

// Q3
type UpdateParams struct {
	Id      int64
	Updates []struct {
		Key   string
		Value string
	}
}

// Q4
type BulkParams struct {
	Orders []struct {
		Name   string
		Amount int64
	}
}

// Q5
type ColumnParams struct {
	Columns []string
}

// Q6
type UnionParams struct {
	Ids []int64
}

// Q7
type CaseParams struct {
	Cases []struct {
		Column    string
		Condition string
		Result    string
	}
}

// Q8
type DynamicParams struct {
	Conditions []struct {
		Key   string
		Value string
	}
	OrderBy []struct {
		Key   string
		Value string
	}
}

// Q9
type SubParams struct {
	Status    *string
	MinAmount *int64
}

// Q10
type GradeParams struct {
	Grade *string
}

// Q11
type DeleteParams struct {
	Ids    []int64
	Status *string
}

// Q12
type AliasParams struct {
	Columns []struct {
		Key   string
		Value string
	}
}

// Q13
type PageParams struct {
	Keyword *string
	Status  *string
	Limit   int64
	Offset  int64
}

// Q14
type JoinParams struct {
	JoinUser     *bool
	JoinProduct  *bool
	JoinCategory *bool
	JoinShipping *bool
	JoinPayment  *bool
	Conditions   []struct {
		Key   string
		Value string
	}
}

// Q15
type GroupParams struct {
	Status   *string
	MinScore *int
	MinCount *int
	MinTotal *int64
}
`

const ordersKX = `
<orders>

  <searchOrders set:{"orders#SearchParams"}>
    SELECT * FROM orders
    [[ if status != nil :{
      <where>
        [[ if ids != nil :{
          AND id IN (
            [[ for id in ids :{
              #{id},
            }]]
          )
        } else if grade != nil :{
          AND grade = #{grade}
        } else :{
          AND 1=1
        }]]
        [[ for filter in filters :{
          AND ${filter.Key} = #{filter.Value},
        }]]
        [[ if keyword != nil :{
          AND name LIKE #{keyword}
        }]]
        [[ if startDate != nil :{
          AND created_at \>= #{startDate}
        }]]
      </>
    }]]
  </>

  <searchOrdersAdvanced set:{"orders#AdvancedParams"}>
    SELECT * FROM orders
    <where>
      [[ if minScore != nil :{
        AND score \>= #{minScore}
      }]]
      [[ if maxScore != nil :{
        AND score \<= #{maxScore}
      }]]
      [[ if minAmount != nil :{
        AND amount \> #{minAmount}
      }]]
      [[ if maxAmount != nil :{
        AND amount \< #{maxAmount}
      }]]
    </>
  </>

  <updateOrderStatus set:{"orders#UpdateParams"}>
    UPDATE orders SET
    [[ for field in updates :{
      ${field.Key} = #{field.Value},
    }]]
    WHERE id = #{id}
  </>

  <bulkInsertOrders set:{"orders#BulkParams"}>
    INSERT INTO orders (name, amount)
    VALUES
    [[ for order in orders :{
      (#{order.Name}, #{order.Amount}),
    }]]
  </>

  <selectOrderColumns set:{"orders#ColumnParams"}>
    SELECT
    [[ for col in columns :{
      ${col},
    }]]
    FROM orders
  </>

  <selectWithUnion set:{"orders#UnionParams"}>
    [[ for id in ids :{
      UNION ALL SELECT * FROM orders WHERE id = #{id}
    }]]
  </>

  <selectCaseWhen set:{"orders#CaseParams"}>
    SELECT id,
    CASE
      [[ for item in cases :{
        WHEN ${item.Column} = #{item.Condition} THEN #{item.Result}
      }]]
    END as label
    FROM orders
  </>

  <selectDynamicOrder set:{"orders#DynamicParams"}>
    SELECT * FROM orders
    <where>
      [[ for cond in conditions :{
        AND ${cond.Key} = #{cond.Value},
      }]]
    </>
    ORDER BY
    [[ for ob in orderBy :{
      ${ob.Key} ${ob.Value},
    }]]
  </>

  <selectSubquery set:{"orders#SubParams"}>
    SELECT * FROM (
      SELECT * FROM orders
      <where>
        [[ if status != nil :{
          AND status = #{status}
        }]]
      </>
    ) sub
    <where>
      [[ if minAmount != nil :{
        AND sub.amount \>= #{minAmount}
      }]]
    </>
  </>

  <selectByGrade set:{"orders#GradeParams"}>
    SELECT * FROM orders
    [[ switch (grade) :{
      case ("A") :{
        AND score \>= 90
      }
      case ("B") :{
        AND score \>= 80
      }
      case ("C") :{
        AND score \>= 70
      }
    }]]
  </>

  <deleteOrders set:{"orders#DeleteParams"}>
    DELETE FROM orders
    <where>
      [[ if ids != nil :{
        AND id IN (
          [[ for id in ids :{
            #{id},
          }]]
        )
      }]]
      [[ if status != nil :{
        AND status = #{status}
      }]]
    </>
  </>

  <selectWithAlias set:{"orders#AliasParams"}>
    SELECT
    [[ for col in columns :{
      ${col.Key} AS ${col.Value},
    }]]
    FROM orders
  </>

  <selectPaged set:{"orders#PageParams"}>
    SELECT * FROM orders
    <where>
      [[ if keyword != nil :{
        AND name LIKE #{keyword}
      }]]
      [[ if status != nil :{
        AND status = #{status}
      }]]
    </>
    LIMIT #{limit} OFFSET #{offset}
  </>

  <selectMultiJoin set:{"orders#JoinParams"}>
    SELECT * FROM orders o
    [[ if joinUser != nil :{
      JOIN users u ON o.user_id = u.id
    } else if joinProduct != nil :{
      JOIN products p ON o.product_id = p.id
    } else if joinCategory != nil :{
      JOIN categories c ON o.category_id = c.id
    } else if joinShipping != nil :{
      JOIN shipping s ON o.shipping_id = s.id
    } else if joinPayment != nil :{
      JOIN payments pay ON o.payment_id = pay.id
    } else :{
      -- no join
    }]]
    <where>
      [[ for cond in conditions :{
        AND ${cond.Key} = #{cond.Value},
      }]]
    </>
  </>

  <selectComplexGroup set:{"orders#GroupParams"}>
    SELECT grade, COUNT(*) as cnt, SUM(amount) as total
    FROM orders
    <where>
      [[ if status != nil :{
        AND status = #{status}
      }]]
      [[ if minScore != nil :{
        AND score \>= #{minScore}
      }]]
    </>
    GROUP BY grade
    [[ if minCount != nil :{
      HAVING COUNT(*) \>= #{minCount}
      [[ if minTotal != nil :{
        AND SUM(amount) \> #{minTotal}
      }]]
    }]]
  </>

</>
`

func setupOrders(t *testing.T) *dokdo.Dokdo {
	t.Helper()
	dir := t.TempDir()
	writeGo(t, dir, "orders.go", ordersGo)
	writeKX(t, dir, "orders.kx", ordersKX)
	return mustLoad(t, dir)
}

// ─────────────────────────────────────────────────────────────────
// Q1 — searchOrders
// ─────────────────────────────────────────────────────────────────

func TestOrders_SearchOrders(t *testing.T) {
	dq := setupOrders(t)

	type SearchParams struct {
		Status    *string
		Ids       []int64
		Grade     *string
		Keyword   *string
		StartDate *string
		Filters   []struct{ Key, Value string }
	}

	t.Run("IdsActive", func(t *testing.T) {
		active := "active"
		keyword := "%kim%"
		startDate := "2024-01-01"
		params := SearchParams{
			Status:    &active,
			Ids:       []int64{1, 2, 3},
			Filters:   []struct{ Key, Value string }{{"category", "online"}},
			Keyword:   &keyword,
			StartDate: &startDate,
		}
		sql, args, err := dq.Build("orders#searchOrders", params)
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		if !strings.Contains(sql, "WHERE") {
			t.Errorf("expected WHERE — sql: %q", sql)
		}
		if !strings.Contains(sql, "IN (") {
			t.Errorf("expected IN ( — sql: %q", sql)
		}
		if !strings.Contains(sql, "LIKE") {
			t.Errorf("expected LIKE — sql: %q", sql)
		}
		if !strings.Contains(sql, "created_at") {
			t.Errorf("expected created_at — sql: %q", sql)
		}
		if got := strings.Count(sql, "?"); got != 6 {
			t.Errorf("? count: got %d, want 6 — sql: %q", got, sql)
		}
		if len(args) != 6 {
			t.Fatalf("args length: got %d, want 6", len(args))
		}
		if args[0] != int64(1) {
			t.Errorf("args[0]: got %v (%T), want int64(1)", args[0], args[0])
		}
		if args[1] != int64(2) {
			t.Errorf("args[1]: got %v (%T), want int64(2)", args[1], args[1])
		}
		if args[2] != int64(3) {
			t.Errorf("args[2]: got %v (%T), want int64(3)", args[2], args[2])
		}
		if args[3] != "online" {
			t.Errorf("args[3]: got %v (%T), want \"online\"", args[3], args[3])
		}
	})

	t.Run("StatusNil", func(t *testing.T) {
		sql, args, err := dq.Build("orders#searchOrders", SearchParams{})
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		if strings.Contains(sql, "WHERE") {
			t.Errorf("WHERE should be absent — sql: %q", sql)
		}
		if got := strings.Count(sql, "?"); got != 0 {
			t.Errorf("? count: got %d, want 0 — sql: %q", got, sql)
		}
		if len(args) != 0 {
			t.Errorf("args length: got %d, want 0", len(args))
		}
	})

	t.Run("GradeBranch", func(t *testing.T) {
		active := "active"
		grade := "VIP"
		params := SearchParams{
			Status:  &active,
			Grade:   &grade,
			Filters: []struct{ Key, Value string }{},
		}
		sql, args, err := dq.Build("orders#searchOrders", params)
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		if !strings.Contains(sql, "WHERE") {
			t.Errorf("expected WHERE — sql: %q", sql)
		}
		if !strings.Contains(sql, "grade = ?") {
			t.Errorf("expected grade = ? — sql: %q", sql)
		}
		if strings.Contains(sql, "IN (") {
			t.Errorf("IN ( should be absent — sql: %q", sql)
		}
		if got := strings.Count(sql, "?"); got != 1 {
			t.Errorf("? count: got %d, want 1 — sql: %q", got, sql)
		}
		_ = args
	})

	t.Run("ElseBranch", func(t *testing.T) {
		active := "active"
		params := SearchParams{
			Status:  &active,
			Filters: []struct{ Key, Value string }{},
		}
		sql, args, err := dq.Build("orders#searchOrders", params)
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		if !strings.Contains(sql, "WHERE") {
			t.Errorf("expected WHERE — sql: %q", sql)
		}
		if !strings.Contains(sql, "1=1") {
			t.Errorf("expected 1=1 — sql: %q", sql)
		}
		if got := strings.Count(sql, "?"); got != 0 {
			t.Errorf("? count: got %d, want 0 — sql: %q", got, sql)
		}
		_ = args
	})
}

// ─────────────────────────────────────────────────────────────────
// Q2 — searchOrdersAdvanced
// ─────────────────────────────────────────────────────────────────

func TestOrders_SearchOrdersAdvanced(t *testing.T) {
	dq := setupOrders(t)

	type AdvancedParams struct {
		MinScore  *int
		MaxScore  *int
		MinAmount *int64
		MaxAmount *int64
	}

	t.Run("AllActive", func(t *testing.T) {
		minScore := 90
		maxScore := 100
		minAmount := int64(1000)
		maxAmount := int64(9999)
		params := AdvancedParams{
			MinScore:  &minScore,
			MaxScore:  &maxScore,
			MinAmount: &minAmount,
			MaxAmount: &maxAmount,
		}
		sql, args, err := dq.Build("orders#searchOrdersAdvanced", params)
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		if !strings.Contains(sql, "score >= ?") {
			t.Errorf("expected score >= ? — sql: %q", sql)
		}
		if !strings.Contains(sql, "score <= ?") {
			t.Errorf("expected score <= ? — sql: %q", sql)
		}
		if !strings.Contains(sql, "amount > ?") {
			t.Errorf("expected amount > ? — sql: %q", sql)
		}
		if !strings.Contains(sql, "amount < ?") {
			t.Errorf("expected amount < ? — sql: %q", sql)
		}
		if got := strings.Count(sql, "?"); got != 4 {
			t.Errorf("? count: got %d, want 4 — sql: %q", got, sql)
		}
		if len(args) != 4 {
			t.Errorf("args length: got %d, want 4", len(args))
		}
	})

	t.Run("AllNil", func(t *testing.T) {
		sql, args, err := dq.Build("orders#searchOrdersAdvanced", AdvancedParams{})
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		if strings.Contains(sql, "WHERE") {
			t.Errorf("WHERE should be absent — sql: %q", sql)
		}
		if got := strings.Count(sql, "?"); got != 0 {
			t.Errorf("? count: got %d, want 0 — sql: %q", got, sql)
		}
		if len(args) != 0 {
			t.Errorf("args length: got %d, want 0", len(args))
		}
	})

	t.Run("MinScoreOnly", func(t *testing.T) {
		minScore := 80
		params := AdvancedParams{MinScore: &minScore}
		sql, args, err := dq.Build("orders#searchOrdersAdvanced", params)
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		if !strings.Contains(sql, "score >= ?") {
			t.Errorf("expected score >= ? — sql: %q", sql)
		}
		if got := strings.Count(sql, "?"); got != 1 {
			t.Errorf("? count: got %d, want 1 — sql: %q", got, sql)
		}
		if len(args) != 1 {
			t.Errorf("args length: got %d, want 1", len(args))
		}
	})
}

// ─────────────────────────────────────────────────────────────────
// Q3 — updateOrderStatus
// ─────────────────────────────────────────────────────────────────

func TestOrders_UpdateOrderStatus(t *testing.T) {
	dq := setupOrders(t)

	type UpdateParams struct {
		Id      int64
		Updates []struct{ Key, Value string }
	}

	t.Run("WithUpdates", func(t *testing.T) {
		params := UpdateParams{
			Id:      int64(1),
			Updates: []struct{ Key, Value string }{{"name", "done"}, {"grade", "A"}},
		}
		sql, args, err := dq.Build("orders#updateOrderStatus", params)
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		if got := strings.Count(sql, "?"); got != 3 {
			t.Errorf("? count: got %d, want 3 — sql: %q", got, sql)
		}
		if len(args) != 3 {
			t.Fatalf("args length: got %d, want 3", len(args))
		}
		if args[0] != "done" {
			t.Errorf("args[0]: got %v (%T), want \"done\"", args[0], args[0])
		}
		if args[1] != "A" {
			t.Errorf("args[1]: got %v (%T), want \"A\"", args[1], args[1])
		}
		if args[2] != int64(1) {
			t.Errorf("args[2]: got %v (%T), want int64(1)", args[2], args[2])
		}
	})

	t.Run("EmptyUpdates", func(t *testing.T) {
		params := UpdateParams{
			Id:      int64(1),
			Updates: []struct{ Key, Value string }{},
		}
		sql, args, err := dq.Build("orders#updateOrderStatus", params)
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		if got := strings.Count(sql, "?"); got != 1 {
			t.Errorf("? count: got %d, want 1 — sql: %q", got, sql)
		}
		if len(args) != 1 {
			t.Fatalf("args length: got %d, want 1", len(args))
		}
		if args[0] != int64(1) {
			t.Errorf("args[0]: got %v (%T), want int64(1)", args[0], args[0])
		}
	})
}

// ─────────────────────────────────────────────────────────────────
// Q4 — bulkInsertOrders
// ─────────────────────────────────────────────────────────────────

func TestOrders_BulkInsertOrders(t *testing.T) {
	dq := setupOrders(t)

	type BulkParams struct {
		Orders []struct {
			Name   string
			Amount int64
		}
	}

	t.Run("ThreeRows", func(t *testing.T) {
		params := BulkParams{
			Orders: []struct {
				Name   string
				Amount int64
			}{
				{"orderA", int64(100)},
				{"orderB", int64(200)},
				{"orderC", int64(300)},
			},
		}
		sql, args, err := dq.Build("orders#bulkInsertOrders", params)
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		if !strings.Contains(sql, "INSERT INTO") {
			t.Errorf("expected INSERT INTO — sql: %q", sql)
		}
		if got := strings.Count(sql, "?"); got != 6 {
			t.Errorf("? count: got %d, want 6 — sql: %q", got, sql)
		}
		if len(args) != 6 {
			t.Fatalf("args length: got %d, want 6", len(args))
		}
		if args[0] != "orderA" {
			t.Errorf("args[0]: got %v, want \"orderA\"", args[0])
		}
		if args[1] != int64(100) {
			t.Errorf("args[1]: got %v (%T), want int64(100)", args[1], args[1])
		}
		if args[2] != "orderB" {
			t.Errorf("args[2]: got %v, want \"orderB\"", args[2])
		}
		if args[3] != int64(200) {
			t.Errorf("args[3]: got %v (%T), want int64(200)", args[3], args[3])
		}
	})

	t.Run("EmptyOrders", func(t *testing.T) {
		params := BulkParams{
			Orders: []struct {
				Name   string
				Amount int64
			}{},
		}
		_, _, err := dq.Build("orders#bulkInsertOrders", params)
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
	})
}

// ─────────────────────────────────────────────────────────────────
// Q5 — selectOrderColumns
// ─────────────────────────────────────────────────────────────────

func TestOrders_SelectOrderColumns(t *testing.T) {
	dq := setupOrders(t)

	type ColumnParams struct {
		Columns []string
	}

	t.Run("ThreeColumns", func(t *testing.T) {
		params := ColumnParams{Columns: []string{"id", "name", "amount"}}
		sql, _, err := dq.Build("orders#selectOrderColumns", params)
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		if !strings.Contains(sql, "id") {
			t.Errorf("expected id — sql: %q", sql)
		}
		if !strings.Contains(sql, "name") {
			t.Errorf("expected name — sql: %q", sql)
		}
		if !strings.Contains(sql, "amount") {
			t.Errorf("expected amount — sql: %q", sql)
		}
		if !strings.Contains(sql, "FROM orders") {
			t.Errorf("expected FROM orders — sql: %q", sql)
		}
		if got := strings.Count(sql, "?"); got != 0 {
			t.Errorf("? count: got %d, want 0 — sql: %q", got, sql)
		}
	})
}

// ─────────────────────────────────────────────────────────────────
// Q6 — selectWithUnion
// ─────────────────────────────────────────────────────────────────

func TestOrders_SelectWithUnion(t *testing.T) {
	dq := setupOrders(t)

	type UnionParams struct {
		Ids []int64
	}

	t.Run("ThreeIds", func(t *testing.T) {
		params := UnionParams{Ids: []int64{10, 20, 30}}
		sql, args, err := dq.Build("orders#selectWithUnion", params)
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		if got := strings.Count(strings.ToUpper(sql), "UNION ALL"); got != 3 {
			t.Errorf("UNION ALL count: got %d, want 3 — sql: %q", got, sql)
		}
		if got := strings.Count(sql, "?"); got != 3 {
			t.Errorf("? count: got %d, want 3 — sql: %q", got, sql)
		}
		if len(args) != 3 {
			t.Fatalf("args length: got %d, want 3", len(args))
		}
		if args[0] != int64(10) {
			t.Errorf("args[0]: got %v (%T), want int64(10)", args[0], args[0])
		}
		if args[1] != int64(20) {
			t.Errorf("args[1]: got %v (%T), want int64(20)", args[1], args[1])
		}
		if args[2] != int64(30) {
			t.Errorf("args[2]: got %v (%T), want int64(30)", args[2], args[2])
		}
	})
}

// ─────────────────────────────────────────────────────────────────
// Q7 — selectCaseWhen
// ─────────────────────────────────────────────────────────────────

func TestOrders_SelectCaseWhen(t *testing.T) {
	dq := setupOrders(t)

	type CaseParams struct {
		Cases []struct{ Column, Condition, Result string }
	}

	t.Run("TwoCases", func(t *testing.T) {
		params := CaseParams{
			Cases: []struct{ Column, Condition, Result string }{
				{"grade", "A", "우수"},
				{"grade", "B", "보통"},
			},
		}
		sql, args, err := dq.Build("orders#selectCaseWhen", params)
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		if !strings.Contains(sql, "WHEN grade = ?") {
			t.Errorf("expected WHEN grade = ? — sql: %q", sql)
		}
		if got := strings.Count(sql, "?"); got != 4 {
			t.Errorf("? count: got %d, want 4 — sql: %q", got, sql)
		}
		if len(args) != 4 {
			t.Fatalf("args length: got %d, want 4", len(args))
		}
		if args[0] != "A" {
			t.Errorf("args[0]: got %v, want \"A\"", args[0])
		}
		if args[1] != "우수" {
			t.Errorf("args[1]: got %v, want \"우수\"", args[1])
		}
		if args[2] != "B" {
			t.Errorf("args[2]: got %v, want \"B\"", args[2])
		}
		if args[3] != "보통" {
			t.Errorf("args[3]: got %v, want \"보통\"", args[3])
		}
	})
}

// ─────────────────────────────────────────────────────────────────
// Q8 — selectDynamicOrder
// ─────────────────────────────────────────────────────────────────

func TestOrders_SelectDynamicOrder(t *testing.T) {
	dq := setupOrders(t)

	type DynamicParams struct {
		Conditions []struct{ Key, Value string }
		OrderBy    []struct{ Key, Value string }
	}

	t.Run("WithCondition", func(t *testing.T) {
		params := DynamicParams{
			Conditions: []struct{ Key, Value string }{{"created_at", "active"}},
			OrderBy:    []struct{ Key, Value string }{{"created_at", "DESC"}},
		}
		sql, args, err := dq.Build("orders#selectDynamicOrder", params)
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		if !strings.Contains(sql, "WHERE") {
			t.Errorf("expected WHERE — sql: %q", sql)
		}
		if !strings.Contains(sql, "ORDER BY") {
			t.Errorf("expected ORDER BY — sql: %q", sql)
		}
		if got := strings.Count(sql, "?"); got != 1 {
			t.Errorf("? count: got %d, want 1 — sql: %q", got, sql)
		}
		if len(args) != 1 {
			t.Fatalf("args length: got %d, want 1", len(args))
		}
		if args[0] != "active" {
			t.Errorf("args[0]: got %v, want \"active\"", args[0])
		}
	})

	t.Run("EmptyConditions", func(t *testing.T) {
		params := DynamicParams{
			Conditions: []struct{ Key, Value string }{},
			OrderBy:    []struct{ Key, Value string }{{"id", "ASC"}},
		}
		sql, _, err := dq.Build("orders#selectDynamicOrder", params)
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		if strings.Contains(sql, "WHERE") {
			t.Errorf("WHERE should be absent — sql: %q", sql)
		}
		if !strings.Contains(sql, "ORDER BY") {
			t.Errorf("expected ORDER BY — sql: %q", sql)
		}
	})
}

// ─────────────────────────────────────────────────────────────────
// Q9 — selectSubquery
// ─────────────────────────────────────────────────────────────────

func TestOrders_SelectSubquery(t *testing.T) {
	dq := setupOrders(t)

	type SubParams struct {
		Status    *string
		MinAmount *int64
	}

	t.Run("BothActive", func(t *testing.T) {
		active := "active"
		minAmount := int64(1000)
		params := SubParams{Status: &active, MinAmount: &minAmount}
		sql, args, err := dq.Build("orders#selectSubquery", params)
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		if !strings.Contains(sql, "status = ?") {
			t.Errorf("expected status = ? — sql: %q", sql)
		}
		if !strings.Contains(sql, "sub.amount") {
			t.Errorf("expected sub.amount — sql: %q", sql)
		}
		if got := strings.Count(sql, "?"); got != 2 {
			t.Errorf("? count: got %d, want 2 — sql: %q", got, sql)
		}
		if len(args) != 2 {
			t.Fatalf("args length: got %d, want 2", len(args))
		}
		if s, ok := args[0].(*string); !ok || *s != "active" {
			t.Errorf("args[0]: got %v (%T), want *string(\"active\")", args[0], args[0])
		}
		if n, ok := args[1].(*int64); !ok || *n != int64(1000) {
			t.Errorf("args[1]: got %v (%T), want *int64(1000)", args[1], args[1])
		}
	})

	t.Run("BothNil", func(t *testing.T) {
		sql, args, err := dq.Build("orders#selectSubquery", SubParams{})
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		if got := strings.Count(sql, "WHERE"); got != 0 {
			t.Errorf("WHERE count: got %d, want 0 — sql: %q", got, sql)
		}
		if got := strings.Count(sql, "?"); got != 0 {
			t.Errorf("? count: got %d, want 0 — sql: %q", got, sql)
		}
		if len(args) != 0 {
			t.Errorf("args length: got %d, want 0", len(args))
		}
	})
}

// ─────────────────────────────────────────────────────────────────
// Q10 — selectByGrade
// ─────────────────────────────────────────────────────────────────

func TestOrders_SelectByGrade(t *testing.T) {
	dq := setupOrders(t)

	type GradeParams struct {
		Grade *string
	}

	t.Run("GradeA", func(t *testing.T) {
		grade := "A"
		sql, _, err := dq.Build("orders#selectByGrade", GradeParams{Grade: &grade})
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		if !strings.Contains(sql, "score >= 90") {
			t.Errorf("expected score >= 90 — sql: %q", sql)
		}
		if strings.Contains(sql, "score >= 80") {
			t.Errorf("score >= 80 should be absent — sql: %q", sql)
		}
		if strings.Contains(sql, "score >= 70") {
			t.Errorf("score >= 70 should be absent — sql: %q", sql)
		}
	})

	t.Run("GradeB", func(t *testing.T) {
		grade := "B"
		sql, _, err := dq.Build("orders#selectByGrade", GradeParams{Grade: &grade})
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		if !strings.Contains(sql, "score >= 80") {
			t.Errorf("expected score >= 80 — sql: %q", sql)
		}
		if strings.Contains(sql, "score >= 90") {
			t.Errorf("score >= 90 should be absent — sql: %q", sql)
		}
	})

	t.Run("GradeD", func(t *testing.T) {
		grade := "D"
		sql, _, err := dq.Build("orders#selectByGrade", GradeParams{Grade: &grade})
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		if strings.Contains(sql, "score") {
			t.Errorf("score should be absent for unmatched grade — sql: %q", sql)
		}
	})
}

// ─────────────────────────────────────────────────────────────────
// Q11 — deleteOrders
// ─────────────────────────────────────────────────────────────────

func TestOrders_DeleteOrders(t *testing.T) {
	dq := setupOrders(t)

	type DeleteParams struct {
		Ids    []int64
		Status *string
	}

	t.Run("AllActive", func(t *testing.T) {
		cancelled := "cancelled"
		params := DeleteParams{Ids: []int64{1, 2, 3}, Status: &cancelled}
		sql, args, err := dq.Build("orders#deleteOrders", params)
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		if !strings.Contains(sql, "WHERE") {
			t.Errorf("expected WHERE — sql: %q", sql)
		}
		if !strings.Contains(sql, "IN (") {
			t.Errorf("expected IN ( — sql: %q", sql)
		}
		if !strings.Contains(sql, "status = ?") {
			t.Errorf("expected status = ? — sql: %q", sql)
		}
		if got := strings.Count(sql, "?"); got != 4 {
			t.Errorf("? count: got %d, want 4 — sql: %q", got, sql)
		}
		if len(args) != 4 {
			t.Fatalf("args length: got %d, want 4", len(args))
		}
		if args[0] != int64(1) {
			t.Errorf("args[0]: got %v (%T), want int64(1)", args[0], args[0])
		}
		if args[1] != int64(2) {
			t.Errorf("args[1]: got %v (%T), want int64(2)", args[1], args[1])
		}
		if args[2] != int64(3) {
			t.Errorf("args[2]: got %v (%T), want int64(3)", args[2], args[2])
		}
	})

	t.Run("IdsNil", func(t *testing.T) {
		cancelled := "cancelled"
		params := DeleteParams{Status: &cancelled}
		sql, args, err := dq.Build("orders#deleteOrders", params)
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		if !strings.Contains(sql, "WHERE") {
			t.Errorf("expected WHERE — sql: %q", sql)
		}
		if strings.Contains(sql, "IN (") {
			t.Errorf("IN ( should be absent — sql: %q", sql)
		}
		if got := strings.Count(sql, "?"); got != 1 {
			t.Errorf("? count: got %d, want 1 — sql: %q", got, sql)
		}
		_ = args
	})

	t.Run("AllNil", func(t *testing.T) {
		sql, args, err := dq.Build("orders#deleteOrders", DeleteParams{})
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		if strings.Contains(sql, "WHERE") {
			t.Errorf("WHERE should be absent — sql: %q", sql)
		}
		if got := strings.Count(sql, "?"); got != 0 {
			t.Errorf("? count: got %d, want 0 — sql: %q", got, sql)
		}
		if len(args) != 0 {
			t.Errorf("args length: got %d, want 0", len(args))
		}
	})
}

// ─────────────────────────────────────────────────────────────────
// Q12 — selectWithAlias
// ─────────────────────────────────────────────────────────────────

func TestOrders_SelectWithAlias(t *testing.T) {
	dq := setupOrders(t)

	type AliasParams struct {
		Columns []struct{ Key, Value string }
	}

	t.Run("TwoAliases", func(t *testing.T) {
		params := AliasParams{
			Columns: []struct{ Key, Value string }{
				{"order_id", "oid"},
				{"order_name", "oname"},
			},
		}
		sql, _, err := dq.Build("orders#selectWithAlias", params)
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		if !strings.Contains(sql, "AS") {
			t.Errorf("expected AS — sql: %q", sql)
		}
		if !strings.Contains(sql, "FROM orders") {
			t.Errorf("expected FROM orders — sql: %q", sql)
		}
		if got := strings.Count(sql, "AS"); got != 2 {
			t.Errorf("AS count: got %d, want 2 — sql: %q", got, sql)
		}
		if got := strings.Count(sql, "?"); got != 0 {
			t.Errorf("? count: got %d, want 0 — sql: %q", got, sql)
		}
	})
}

// ─────────────────────────────────────────────────────────────────
// Q13 — selectPaged
// ─────────────────────────────────────────────────────────────────

func TestOrders_SelectPaged(t *testing.T) {
	dq := setupOrders(t)

	type PageParams struct {
		Keyword *string
		Status  *string
		Limit   int64
		Offset  int64
	}

	t.Run("WithWhere", func(t *testing.T) {
		keyword := "%kim%"
		active := "active"
		params := PageParams{
			Keyword: &keyword,
			Status:  &active,
			Limit:   int64(10),
			Offset:  int64(0),
		}
		sql, args, err := dq.Build("orders#selectPaged", params)
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		if !strings.Contains(sql, "WHERE") {
			t.Errorf("expected WHERE — sql: %q", sql)
		}
		if !strings.Contains(sql, "LIMIT") {
			t.Errorf("expected LIMIT — sql: %q", sql)
		}
		if !strings.Contains(sql, "OFFSET") {
			t.Errorf("expected OFFSET — sql: %q", sql)
		}
		if got := strings.Count(sql, "?"); got != 4 {
			t.Errorf("? count: got %d, want 4 — sql: %q", got, sql)
		}
		if len(args) != 4 {
			t.Fatalf("args length: got %d, want 4", len(args))
		}
		if args[2] != int64(10) {
			t.Errorf("args[2]: got %v (%T), want int64(10)", args[2], args[2])
		}
		if args[3] != int64(0) {
			t.Errorf("args[3]: got %v (%T), want int64(0)", args[3], args[3])
		}
	})

	t.Run("WhereNil", func(t *testing.T) {
		params := PageParams{Limit: int64(10), Offset: int64(20)}
		sql, args, err := dq.Build("orders#selectPaged", params)
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		if strings.Contains(sql, "WHERE") {
			t.Errorf("WHERE should be absent — sql: %q", sql)
		}
		if !strings.Contains(sql, "LIMIT") {
			t.Errorf("expected LIMIT — sql: %q", sql)
		}
		if got := strings.Count(sql, "?"); got != 2 {
			t.Errorf("? count: got %d, want 2 — sql: %q", got, sql)
		}
		if len(args) != 2 {
			t.Fatalf("args length: got %d, want 2", len(args))
		}
		if args[0] != int64(10) {
			t.Errorf("args[0]: got %v (%T), want int64(10)", args[0], args[0])
		}
		if args[1] != int64(20) {
			t.Errorf("args[1]: got %v (%T), want int64(20)", args[1], args[1])
		}
	})
}

// ─────────────────────────────────────────────────────────────────
// Q14 — selectMultiJoin
// ─────────────────────────────────────────────────────────────────

func TestOrders_SelectMultiJoin(t *testing.T) {
	dq := setupOrders(t)

	type JoinParams struct {
		JoinUser     *bool
		JoinProduct  *bool
		JoinCategory *bool
		JoinShipping *bool
		JoinPayment  *bool
		Conditions   []struct{ Key, Value string }
	}

	bTrue := true

	t.Run("JoinUser", func(t *testing.T) {
		params := JoinParams{
			JoinUser:   &bTrue,
			Conditions: []struct{ Key, Value string }{{"o.status", "active"}},
		}
		sql, args, err := dq.Build("orders#selectMultiJoin", params)
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		if !strings.Contains(sql, "JOIN users u") {
			t.Errorf("expected JOIN users u — sql: %q", sql)
		}
		if !strings.Contains(sql, "WHERE") {
			t.Errorf("expected WHERE — sql: %q", sql)
		}
		if got := strings.Count(sql, "?"); got != 1 {
			t.Errorf("? count: got %d, want 1 — sql: %q", got, sql)
		}
		if len(args) != 1 {
			t.Fatalf("args length: got %d, want 1", len(args))
		}
		if args[0] != "active" {
			t.Errorf("args[0]: got %v, want \"active\"", args[0])
		}
	})

	t.Run("JoinPayment", func(t *testing.T) {
		params := JoinParams{
			JoinPayment: &bTrue,
			Conditions:  []struct{ Key, Value string }{},
		}
		sql, _, err := dq.Build("orders#selectMultiJoin", params)
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		if !strings.Contains(sql, "JOIN payments pay") {
			t.Errorf("expected JOIN payments pay — sql: %q", sql)
		}
		if strings.Contains(sql, "WHERE") {
			t.Errorf("WHERE should be absent — sql: %q", sql)
		}
	})

	t.Run("NoJoin", func(t *testing.T) {
		params := JoinParams{
			Conditions: []struct{ Key, Value string }{},
		}
		sql, _, err := dq.Build("orders#selectMultiJoin", params)
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		if !strings.Contains(sql, "-- no join") {
			t.Errorf("expected -- no join — sql: %q", sql)
		}
		if strings.Contains(sql, "WHERE") {
			t.Errorf("WHERE should be absent — sql: %q", sql)
		}
	})
}

// ─────────────────────────────────────────────────────────────────
// Q15 — selectComplexGroup
// ─────────────────────────────────────────────────────────────────

func TestOrders_SelectComplexGroup(t *testing.T) {
	dq := setupOrders(t)

	type GroupParams struct {
		Status   *string
		MinScore *int
		MinCount *int
		MinTotal *int64
	}

	t.Run("AllActive", func(t *testing.T) {
		active := "active"
		minScore := 80
		minCount := 5
		minTotal := int64(10000)
		params := GroupParams{
			Status:   &active,
			MinScore: &minScore,
			MinCount: &minCount,
			MinTotal: &minTotal,
		}
		sql, args, err := dq.Build("orders#selectComplexGroup", params)
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		if !strings.Contains(sql, "WHERE") {
			t.Errorf("expected WHERE — sql: %q", sql)
		}
		if !strings.Contains(sql, "GROUP BY") {
			t.Errorf("expected GROUP BY — sql: %q", sql)
		}
		if !strings.Contains(sql, "HAVING") {
			t.Errorf("expected HAVING — sql: %q", sql)
		}
		if !strings.Contains(sql, "SUM(amount)") {
			t.Errorf("expected SUM(amount) — sql: %q", sql)
		}
		if got := strings.Count(sql, "?"); got != 4 {
			t.Errorf("? count: got %d, want 4 — sql: %q", got, sql)
		}
		if len(args) != 4 {
			t.Errorf("args length: got %d, want 4", len(args))
		}
	})

	t.Run("NoHaving", func(t *testing.T) {
		active := "active"
		params := GroupParams{Status: &active}
		sql, args, err := dq.Build("orders#selectComplexGroup", params)
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		if !strings.Contains(sql, "WHERE") {
			t.Errorf("expected WHERE — sql: %q", sql)
		}
		if !strings.Contains(sql, "GROUP BY") {
			t.Errorf("expected GROUP BY — sql: %q", sql)
		}
		if strings.Contains(sql, "HAVING") {
			t.Errorf("HAVING should be absent — sql: %q", sql)
		}
		if got := strings.Count(sql, "?"); got != 1 {
			t.Errorf("? count: got %d, want 1 — sql: %q", got, sql)
		}
		if len(args) != 1 {
			t.Errorf("args length: got %d, want 1", len(args))
		}
	})
}
