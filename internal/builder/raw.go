package builder

import (
	"regexp"
	"strings"
)

const maxIdentifierLen = 128

var rawPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_.\-]*$`)

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

var splitOnSeparator = func(r rune) bool { return r == '.' || r == '-' }

// validateRawInner validates the content inside backtick-quoted identifiers.
// Does NOT check allowlist (backtick quoting is the caller's explicit opt-in for reserved words).
// Does NOT check length (caller already verified).
func validateRawInner(inner string) error {
	for _, r := range inner {
		if r > 0x7F {
			return &RuntimeError{Message: "non-ASCII character in identifier: " + inner}
		}
	}
	if !rawPattern.MatchString(inner) {
		return &RuntimeError{Message: "invalid identifier: " + inner}
	}
	if strings.Contains(inner, "--") {
		return &RuntimeError{Message: "consecutive hyphens not allowed: " + inner}
	}
	for _, seg := range strings.FieldsFunc(inner, splitOnSeparator) {
		if blocklist[strings.ToUpper(seg)] {
			return &RuntimeError{Message: "blocked SQL keyword: " + inner}
		}
	}
	return nil
}

func validateRaw(val string) (string, error) {
	val = strings.TrimSpace(val)

	// E: 길이 제한
	if len(val) > maxIdentifierLen {
		return "", &RuntimeError{Message: "identifier too long"}
	}

	// B step1: 백틱 개수 1차 확인
	btCount := strings.Count(val, "`")

	if btCount == 0 {
		// D: 비ASCII 거부
		for _, r := range val {
			if r > 0x7F {
				return "", &RuntimeError{Message: "non-ASCII character in identifier: " + val}
			}
		}
		// 형식 검증
		if !rawPattern.MatchString(val) {
			return "", &RuntimeError{Message: "invalid identifier: " + val}
		}
		if strings.Contains(val, "--") {
			return "", &RuntimeError{Message: "consecutive hyphens not allowed: " + val}
		}
		// A+C: 세그먼트별 blocklist 검사 ('.' 및 '-' 기준 분리)
		for _, seg := range strings.FieldsFunc(val, splitOnSeparator) {
			if blocklist[strings.ToUpper(seg)] {
				return "", &RuntimeError{Message: "blocked SQL keyword: " + val}
			}
		}
		// allowlist: 전체 값 기준으로만 검사 (세그먼트 분리 안 함)
		// — t.column처럼 한정자가 붙으면 전체 문자열이 allowlist에 없으므로 통과
		if allowlist[strings.ToUpper(val)] {
			return "", &RuntimeError{Message: "reserved word requires backtick: `" + val + "`"}
		}
		return val, nil
	}

	// B step2: 정확히 2개여야 함
	if btCount != 2 {
		return "", &RuntimeError{Message: "invalid backtick usage"}
	}
	// B step3: 앞뒤 대칭 확인
	if !strings.HasPrefix(val, "`") || !strings.HasSuffix(val, "`") {
		return "", &RuntimeError{Message: "invalid backtick usage"}
	}
	// len == 2이면 `` (빈 내부)
	if len(val) < 3 {
		return "", &RuntimeError{Message: "empty backtick identifier"}
	}
	inner := val[1 : len(val)-1]
	// 방어적 이중 확인 (btCount==2 + 앞뒤 대칭이면 inner에 백틱 없음이 보장되지만 명시)
	if strings.Contains(inner, "`") {
		return "", &RuntimeError{Message: "invalid backtick usage"}
	}

	if err := validateRawInner(inner); err != nil {
		return "", err
	}
	return val, nil
}
