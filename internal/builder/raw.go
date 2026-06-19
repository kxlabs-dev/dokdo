package builder

import (
	"regexp"
	"strings"
)

var rawPattern = regexp.MustCompile(`^[a-zA-Z0-9_.\-]+$`)

var blocklist = map[string]bool{
	"SELECT": true, "FROM": true, "WHERE": true,
	"JOIN": true, "LEFT": true, "RIGHT": true,
	"INNER": true, "OUTER": true, "FULL": true, "CROSS": true,
	"UNION": true, "INTERSECT": true, "EXCEPT": true,
	"INSERT": true, "UPDATE": true, "DELETE": true,
	"DROP": true, "CREATE": true, "ALTER": true, "TRUNCATE": true,
	"GRANT": true, "REVOKE": true, "COMMIT": true,
	"ROLLBACK": true, "TRANSACTION": true, "BEGIN": true,
	"WITH": true, "HAVING": true, "DISTINCT": true,
}

var allowlist = map[string]bool{
	"IN": true, "IS": true, "NOT": true, "ALL": true,
	"ANY": true, "EXISTS": true,
	"MIN": true, "MAX": true, "SUM": true, "AVG": true, "COUNT": true,
	"ORDER": true, "GROUP": true, "INDEX": true, "KEY": true, "VALUE": true,
	"LIMIT": true, "OFFSET": true, "RANK": true, "LEVEL": true, "ROLE": true,
	"DATE": true, "TIME": true, "TIMESTAMP": true, "TYPE": true, "STATUS": true,
	"DEFAULT": true, "CHECK": true, "SET": true, "LIKE": true,
	"CASE": true, "END": true,
	"PRIMARY": true, "UNIQUE": true, "COLUMN": true,
	"FOREIGN": true, "REFERENCES": true, "CONSTRAINT": true,
}

func validateRaw(val string) (string, error) {
	val = strings.TrimSpace(val)

	if strings.HasPrefix(val, "`") && strings.HasSuffix(val, "`") {
		inner := val[1 : len(val)-1]
		upper := strings.ToUpper(strings.TrimSpace(inner))
		if blocklist[upper] {
			return "", &RuntimeError{Message: "blocked SQL keyword: " + inner}
		}
		return val, nil
	}

	if !rawPattern.MatchString(val) {
		return "", &RuntimeError{Message: "invalid identifier: " + val}
	}

	upper := strings.ToUpper(val)

	if blocklist[upper] {
		return "", &RuntimeError{Message: "blocked SQL keyword: " + val}
	}

	if allowlist[upper] {
		return "", &RuntimeError{Message: "reserved word requires backtick: `" + val + "`"}
	}

	return val, nil
}
