package builder

import (
	"reflect"
	"strings"

	"github.com/kxlabs-dev/dokdo/parser"
)

func (b *Builder) buildForList(n *parser.ForNode, rv reflect.Value, params interface{}) error {
	var parts []string
	var args []interface{}

	for i := 0; i < rv.Len(); i++ {
		item := rv.Index(i).Interface()
		buf := &Builder{}
		if err := buf.build(n.Body, itemScope(n.ItemVar, item, params), true); err != nil {
			return err
		}
		parts = append(parts, strings.TrimSpace(buf.sql.String()))
		args = append(args, buf.args...)
	}

	result := removeTrailingComma(strings.Join(parts, " "))
	b.sql.WriteString(result)
	b.args = append(b.args, args...)
	return nil
}

func itemScope(itemVar string, item interface{}, params interface{}) map[string]interface{} {
	m := paramsToMap(params)
	m[itemVar] = item
	return m
}

func removeTrailingComma(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, ",") {
		s = s[:len(s)-1]
	}
	return s
}
