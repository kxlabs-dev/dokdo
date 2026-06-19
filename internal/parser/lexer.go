package parser

import "strings"

type TokenType int

const (
	TAG_OPEN    TokenType = iota
	TAG_CLOSE             // </>
	PARAM_REF             // set:{ paramName }
	EXPR_OPEN             // [[
	EXPR_CLOSE            // ]]
	BIND                  // #{ path }
	RAW                   // ${ path }
	SQL_TEXT
	WHERE_TAG   // <where>
	ESCAPE      // \< \> \<= \>= !=
	BLOCK_CLOSE // }
	EOF
)

type Token struct {
	Type  TokenType
	Value string
	Line  int
}

type Lexer struct {
	input   string
	pos     int
	line    int
	file    string
	pending []Token
	err     *ParseError
}

func NewLexer(input, file string) *Lexer {
	return &Lexer{input: input, pos: 0, line: 1, file: file}
}

func (l *Lexer) Err() *ParseError {
	return l.err
}

// --- cursor helpers ---

func (l *Lexer) peek() byte {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

func (l *Lexer) peek2() byte {
	if l.pos+1 >= len(l.input) {
		return 0
	}
	return l.input[l.pos+1]
}

func (l *Lexer) advance() byte {
	if l.pos >= len(l.input) {
		return 0
	}
	ch := l.input[l.pos]
	l.pos++
	if ch == '\n' {
		l.line++
	}
	return ch
}

func (l *Lexer) advanceN(n int) {
	for i := 0; i < n; i++ {
		l.advance()
	}
}

func (l *Lexer) atEOF() bool {
	return l.pos >= len(l.input)
}

func (l *Lexer) startsWith(s string) bool {
	return strings.HasPrefix(l.input[l.pos:], s)
}

func (l *Lexer) skipWS() {
	for !l.atEOF() && (l.input[l.pos] == ' ' || l.input[l.pos] == '\t') {
		l.pos++
	}
}

func (l *Lexer) skipWSNL() {
	for !l.atEOF() {
		ch := l.input[l.pos]
		if ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n' {
			l.advance()
		} else {
			break
		}
	}
}

func (l *Lexer) makeError(msg string) *ParseError {
	return &ParseError{File: l.file, Line: l.line, Message: msg}
}

func (l *Lexer) readTagName() string {
	start := l.pos
	for !l.atEOF() {
		ch := l.input[l.pos]
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == '_' {
			l.pos++
		} else {
			break
		}
	}
	return l.input[start:l.pos]
}

func (l *Lexer) readParamPath() string {
	start := l.pos
	for !l.atEOF() {
		ch := l.input[l.pos]
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == '_' || ch == '.' {
			l.pos++
		} else {
			break
		}
	}
	return l.input[start:l.pos]
}

func (l *Lexer) consumeTagClose() {
	l.skipWSNL()
	if l.atEOF() || l.input[l.pos] != '>' {
		l.err = l.makeError("expected '>' to close tag")
		return
	}
	l.pos++
}

// --- sub-readers ---

func (l *Lexer) readComment() Token {
	l.advanceN(4) // consume <!--
	for !l.atEOF() {
		if l.startsWith("-->") {
			l.advanceN(3)
			return l.Next()
		}
		l.advance()
	}
	l.err = l.makeError("unterminated comment")
	return Token{Type: EOF, Line: l.line}
}

func (l *Lexer) readBind() Token {
	startLine := l.line
	l.advanceN(2) // consume #{
	l.skipWSNL()
	path := l.readParamPath()
	l.skipWSNL()
	if path == "" {
		l.err = l.makeError("empty #{} bind parameter")
		return Token{Type: EOF, Line: l.line}
	}
	if l.atEOF() || l.input[l.pos] != '}' {
		l.err = l.makeError("expected '}' to close #{}")
		return Token{Type: EOF, Line: l.line}
	}
	l.pos++ // consume }
	return Token{Type: BIND, Value: path, Line: startLine}
}

func (l *Lexer) readRaw() Token {
	startLine := l.line
	l.advanceN(2) // consume ${
	l.skipWSNL()
	path := l.readParamPath()
	l.skipWSNL()
	if path == "" {
		l.err = l.makeError("empty ${} raw parameter")
		return Token{Type: EOF, Line: l.line}
	}
	if l.atEOF() || l.input[l.pos] != '}' {
		l.err = l.makeError("expected '}' to close ${}")
		return Token{Type: EOF, Line: l.line}
	}
	l.pos++ // consume }
	return Token{Type: RAW, Value: path, Line: startLine}
}

func (l *Lexer) readEscape() Token {
	startLine := l.line
	l.advance() // consume \
	ch := l.peek()
	switch ch {
	case '<':
		l.advance() // consume <
		next := l.peek()
		if next == '=' {
			l.advance()
			return Token{Type: ESCAPE, Value: "<=", Line: startLine}
		}
		return Token{Type: ESCAPE, Value: "<", Line: startLine}
	case '>':
		l.advance() // consume >
		next := l.peek()
		if next == '=' {
			l.advance()
			return Token{Type: ESCAPE, Value: ">=", Line: startLine}
		}
		return Token{Type: ESCAPE, Value: ">", Line: startLine}
	default:
		return Token{Type: SQL_TEXT, Value: "\\", Line: startLine}
	}
}

func (l *Lexer) readParamRef() (string, *ParseError) {
	l.advanceN(5) // consume "set:{"
	l.skipWSNL()
	if l.atEOF() || l.input[l.pos] != '"' {
		return "", l.makeError("expected '\"' after 'set:{'")
	}
	l.pos++ // consume opening "
	start := l.pos
	for !l.atEOF() {
		ch := l.input[l.pos]
		if ch == '"' {
			break
		}
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == '_' || ch == '.' || ch == '#' || ch == '-' {
			l.pos++
		} else {
			return "", l.makeError("invalid character in set:{} param reference")
		}
	}
	if l.atEOF() || l.input[l.pos] != '"' {
		return "", l.makeError("unterminated string in set:{}")
	}
	value := l.input[start:l.pos]
	l.pos++ // consume closing "
	if value == "" {
		return "", l.makeError("empty set:{} param reference")
	}
	l.skipWSNL()
	if l.atEOF() || l.input[l.pos] != '}' {
		return "", l.makeError("expected '}' to close set:{}")
	}
	l.pos++ // consume }
	return value, nil
}

func (l *Lexer) readTag() Token {
	startLine := l.line
	l.advance() // consume <

	if l.startsWith("/>") {
		l.advanceN(2)
		return Token{Type: TAG_CLOSE, Value: "</>", Line: startLine}
	}

	name := l.readTagName()
	if name == "" {
		l.err = l.makeError("expected tag name after '<'")
		return Token{Type: EOF, Line: l.line}
	}

	upper := strings.ToUpper(name)
	if upper == "WHERE" {
		l.consumeTagClose()
		if l.err != nil {
			return Token{Type: EOF, Line: l.line}
		}
		return Token{Type: WHERE_TAG, Value: name, Line: startLine}
	}
	l.skipWSNL()
	if l.startsWith("set:{") {
		paramLine := l.line
		paramName, err := l.readParamRef()
		if err != nil {
			l.err = err
			return Token{Type: EOF, Line: l.line}
		}
		l.pending = append(l.pending, Token{Type: PARAM_REF, Value: paramName, Line: paramLine})
	}
	l.consumeTagClose()
	if l.err != nil {
		return Token{Type: EOF, Line: l.line}
	}
	return Token{Type: TAG_OPEN, Value: name, Line: startLine}
}

func (l *Lexer) readSQLText() Token {
	startLine := l.line
	var buf strings.Builder
	for !l.atEOF() {
		if l.startsWith("[[") || l.startsWith("]]") {
			break
		}
		if l.startsWith("#{") || l.startsWith("${") {
			break
		}
		if l.peek() == '\\' && (l.peek2() == '<' || l.peek2() == '>') {
			break
		}
		if l.startsWith("!=") {
			break
		}
		if l.peek() == '<' {
			break
		}
		if l.peek() == '}' {
			break
		}
		buf.WriteByte(l.advance())
	}
	return Token{Type: SQL_TEXT, Value: buf.String(), Line: startLine}
}

// --- main entry point ---

func (l *Lexer) Next() Token {
	if len(l.pending) > 0 {
		tok := l.pending[0]
		l.pending = l.pending[1:]
		return tok
	}
	if l.err != nil {
		return Token{Type: EOF, Line: l.line}
	}
	if l.atEOF() {
		return Token{Type: EOF, Line: l.line}
	}

	switch {
	case l.startsWith("<!--"):
		return l.readComment()
	case l.startsWith("[["):
		startLine := l.line
		l.advanceN(2)
		return Token{Type: EXPR_OPEN, Value: "[[", Line: startLine}
	case l.startsWith("]]"):
		startLine := l.line
		l.advanceN(2)
		return Token{Type: EXPR_CLOSE, Value: "]]", Line: startLine}
	case l.startsWith("#{"):
		return l.readBind()
	case l.startsWith("${"):
		return l.readRaw()
	case l.peek() == '\\':
		return l.readEscape()
	case l.startsWith("!="):
		startLine := l.line
		l.advanceN(2)
		return Token{Type: ESCAPE, Value: "<>", Line: startLine}
	case l.peek() == '}':
		startLine := l.line
		l.advance()
		return Token{Type: BLOCK_CLOSE, Value: "}", Line: startLine}
	case l.peek() == '<':
		return l.readTag()
	default:
		return l.readSQLText()
	}
}
