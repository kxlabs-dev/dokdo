package builder

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode"

	"github.com/kxlabs-dev/dokdo/internal/parser"
)

type Builder struct {
	sql  strings.Builder
	args []interface{}
}

func (b *Builder) build(nodes []parser.SQLNode, params interface{}, inFor bool) error {
	for _, node := range nodes {
		switch n := node.(type) {
		case *parser.SQLText:
			b.sql.WriteString(n.Text)
		case *parser.BindParam:
			val, err := resolveValue(n.Path, params)
			if err != nil {
				return err
			}
			b.sql.WriteString("?")
			b.args = append(b.args, val)
		case *parser.RawParam:
			val, err := resolveValue(n.Path, params)
			if err != nil {
				return err
			}
			s, err := validateRaw(fmt.Sprintf("%v", val))
			if err != nil {
				return err
			}
			b.sql.WriteString(s)
		case *parser.WhereNode:
			if err := b.buildWhere(n, params); err != nil {
				return err
			}
		case *parser.IfNode:
			if err := b.buildIf(n, params, inFor); err != nil {
				return err
			}
		case *parser.SwitchNode:
			if err := b.buildSwitch(n, params, inFor); err != nil {
				return err
			}
		case *parser.ForNode:
			if err := b.buildFor(n, params); err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *Builder) buildIf(n *parser.IfNode, params interface{}, inFor bool) error {
	ok, err := evalCond(n.Cond, params)
	if err != nil {
		return err
	}
	if ok {
		return b.build(n.Then, params, inFor)
	}
	for _, ei := range n.ElseIfs {
		ok, err := evalCond(ei.Cond, params)
		if err != nil {
			return err
		}
		if ok {
			return b.build(ei.Body, params, inFor)
		}
	}
	if n.Else != nil {
		return b.build(n.Else, params, inFor)
	}
	return nil
}

func (b *Builder) buildSwitch(n *parser.SwitchNode, params interface{}, inFor bool) error {
	val, err := resolveValue(n.Expr, params)
	if err != nil {
		return err
	}
	rv := reflect.ValueOf(val)
	for rv.IsValid() && rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			val = nil
			break
		}
		rv = rv.Elem()
		val = rv.Interface()
	}
	strVal := fmt.Sprintf("%v", val)
	for _, c := range n.Cases {
		if strVal == c.Value {
			return b.build(c.Body, params, inFor)
		}
	}
	if n.Default != nil {
		return b.build(n.Default, params, inFor)
	}
	return nil
}

func (b *Builder) buildFor(n *parser.ForNode, params interface{}) error {
	rv, err := resolveSlice(n.Collection, params)
	if err != nil {
		return err
	}
	if rv.Kind() != reflect.Slice {
		return &RuntimeError{Message: fmt.Sprintf("'%s' is not a slice", n.Collection)}
	}
	return b.buildForList(n, rv, params)
}

// evalCond evaluates a binary condition string against params.
// Operators scanned longest-first: <>, >=, <=, ==, >, <
func evalCond(cond string, params interface{}) (bool, error) {
	ops := []string{"<>", ">=", "<=", "==", ">", "<"}
	var lhsStr, op, rhsStr string
	found := false
	for _, candidate := range ops {
		idx := strings.Index(cond, candidate)
		if idx >= 0 {
			lhsStr = strings.TrimSpace(cond[:idx])
			op = candidate
			rhsStr = strings.TrimSpace(cond[idx+len(candidate):])
			found = true
			break
		}
	}
	if !found {
		return false, &RuntimeError{Message: "invalid condition: " + cond}
	}

	lhsRaw, err := resolveValue(lhsStr, params)
	if err != nil {
		return false, err
	}

	if rhsStr == "nil" {
		rv := reflect.ValueOf(lhsRaw)
		var isNil bool
		if !rv.IsValid() {
			isNil = true
		} else {
			switch rv.Kind() {
			case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map,
				reflect.Chan, reflect.Func:
				isNil = rv.IsNil()
			default:
				isNil = false
			}
		}
		switch op {
		case "==":
			return isNil, nil
		case "<>":
			return !isNil, nil
		default:
			return false, &RuntimeError{Message: "operator " + op + " not valid for nil comparison"}
		}
	}

	lhsVal := lhsRaw
	rv := reflect.ValueOf(lhsVal)
	for rv.IsValid() && rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return false, nil
		}
		rv = rv.Elem()
		lhsVal = rv.Interface()
	}

	rhsNum, rhsErr := strconv.ParseFloat(rhsStr, 64)
	if rhsErr == nil {
		lhsNum, convErr := toFloat64(lhsVal)
		if convErr != nil {
			return false, convErr
		}
		switch op {
		case ">":
			return lhsNum > rhsNum, nil
		case "<":
			return lhsNum < rhsNum, nil
		case ">=":
			return lhsNum >= rhsNum, nil
		case "<=":
			return lhsNum <= rhsNum, nil
		case "==":
			return lhsNum == rhsNum, nil
		case "<>":
			return lhsNum != rhsNum, nil
		}
	}

	lhsStr2 := fmt.Sprintf("%v", lhsVal)
	switch op {
	case "==":
		return lhsStr2 == rhsStr, nil
	case "<>":
		return lhsStr2 != rhsStr, nil
	}

	return false, &RuntimeError{Message: fmt.Sprintf("cannot compare %q %s %q", lhsStr, op, rhsStr)}
}

func toFloat64(v interface{}) (float64, error) {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(rv.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(rv.Uint()), nil
	case reflect.Float32, reflect.Float64:
		return rv.Float(), nil
	default:
		return 0, &RuntimeError{Message: fmt.Sprintf("cannot convert %T to number", v)}
	}
}

func resolveValue(path string, params interface{}) (interface{}, error) {
	parts := strings.Split(path, ".")
	rv := reflect.ValueOf(params)
	for _, part := range parts {
		for rv.IsValid() && rv.Kind() == reflect.Ptr {
			if rv.IsNil() {
				return nil, nil
			}
			rv = rv.Elem()
		}
		if !rv.IsValid() {
			return nil, nil
		}
		switch rv.Kind() {
		case reflect.Struct:
			rv = rv.FieldByName(titleCase(part))
			if !rv.IsValid() {
				return nil, &RuntimeError{Message: fmt.Sprintf("field %q not found", part)}
			}
		case reflect.Map:
			mv := rv.MapIndex(reflect.ValueOf(part))
			if !mv.IsValid() {
				return nil, &RuntimeError{Message: fmt.Sprintf("key %q not found in map", part)}
			}
			rv = mv
			if rv.Kind() == reflect.Interface {
				rv = rv.Elem()
			}
		default:
			return nil, &RuntimeError{Message: fmt.Sprintf("cannot resolve %q on %s", part, rv.Kind())}
		}
	}
	if !rv.IsValid() {
		return nil, nil
	}
	return rv.Interface(), nil
}

func resolveSlice(path string, params interface{}) (reflect.Value, error) {
	val, err := resolveValue(path, params)
	if err != nil {
		return reflect.Value{}, err
	}
	rv := reflect.ValueOf(val)
	for rv.IsValid() && rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	return rv, nil
}

func paramsToMap(params interface{}) map[string]interface{} {
	if params == nil {
		return make(map[string]interface{})
	}
	if m, ok := params.(map[string]interface{}); ok {
		result := make(map[string]interface{}, len(m))
		for k, v := range m {
			result[k] = v
		}
		return result
	}
	rv := reflect.ValueOf(params)
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return make(map[string]interface{})
		}
		rv = rv.Elem()
	}
	result := make(map[string]interface{})
	if rv.Kind() != reflect.Struct {
		return result
	}
	t := rv.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		result[lowerFirst(f.Name)] = rv.Field(i).Interface()
	}
	return result
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

func Execute(nodes []parser.SQLNode, params interface{}, info *TypeInfo) (string, []interface{}, error) {
	if info != nil {
		if err := ValidateParams(params, info); err != nil {
			return "", nil, err
		}
	}
	b := &Builder{}
	if err := b.build(nodes, params, false); err != nil {
		return "", nil, err
	}
	return b.sql.String(), b.args, nil
}
