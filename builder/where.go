package builder

import (
	"strings"

	"github.com/kxlabs-dev/dokdo/parser"
)

func (b *Builder) buildWhere(n *parser.WhereNode, params interface{}) error {
	buf := &Builder{}
	if err := buf.build(n.Children, params, false); err != nil {
		return err
	}
	result := strings.TrimSpace(buf.sql.String())
	if result == "" {
		return nil
	}
	result = trimLeadingAndOr(result)
	b.sql.WriteString(" WHERE ")
	b.sql.WriteString(result)
	b.args = append(b.args, buf.args...)
	return nil
}

func trimLeadingAndOr(s string) string {
	upper := strings.ToUpper(s)
	if strings.HasPrefix(upper, "AND ") {
		return s[4:]
	}
	if strings.HasPrefix(upper, "OR ") {
		return s[3:]
	}
	return s
}
