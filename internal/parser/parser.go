package parser

import (
	"fmt"
	"path/filepath"
	"strings"
)

var sqlKeywords = map[string]bool{
	"SELECT": true, "FROM": true, "WHERE": true,
	"JOIN": true, "LEFT": true, "RIGHT": true, "INNER": true,
	"OUTER": true, "FULL": true, "CROSS": true,
	"UNION": true, "INTERSECT": true, "EXCEPT": true,
	"INSERT": true, "UPDATE": true, "DELETE": true,
	"DROP": true, "CREATE": true, "ALTER": true, "TRUNCATE": true,
	"GRANT": true, "REVOKE": true, "COMMIT": true, "ROLLBACK": true,
	"TRANSACTION": true, "BEGIN": true, "WITH": true,
	"HAVING": true, "DISTINCT": true,
}

type Parser struct {
	lexer *Lexer
	cur   Token
	peek  Token
	file  string
}

func newParser(input, file string) *Parser {
	l := NewLexer(input, file)
	p := &Parser{lexer: l, file: file}
	p.cur = l.Next()
	p.peek = l.Next()
	return p
}

func (p *Parser) advance() {
	p.cur = p.peek
	p.peek = p.lexer.Next()
}

func (p *Parser) makeError(line int, msg string) *ParseError {
	return &ParseError{File: p.file, Line: line, Message: msg}
}

func (p *Parser) checkLexErr() error {
	if err := p.lexer.Err(); err != nil {
		return err
	}
	return nil
}

// ParseFile parses a .kx file and returns its AST.
func ParseFile(input, file string) (*QueryFile, error) {
	p := newParser(input, file)
	return p.parseFile()
}

func (p *Parser) parseFile() (*QueryFile, error) {
	qf := &QueryFile{}

	for p.cur.Type == SQL_TEXT {
		p.advance()
	}

	if p.cur.Type != TAG_OPEN {
		return nil, p.makeError(p.cur.Line, "expected namespace root tag")
	}

	namespace := p.cur.Value
	namespaceLine := p.cur.Line
	baseName := strings.TrimSuffix(filepath.Base(p.file), filepath.Ext(p.file))
	if namespace != baseName {
		return nil, p.makeError(namespaceLine, fmt.Sprintf("root tag '<%s>' does not match filename '%s'", namespace, baseName))
	}
	qf.Namespace = namespace
	p.advance()

	for p.cur.Type != TAG_CLOSE && p.cur.Type != EOF {
		switch p.cur.Type {
		case SQL_TEXT:
			p.advance()

		case TAG_OPEN:
			tagName := p.cur.Value
			tagLine := p.cur.Line
			upper := strings.ToUpper(tagName)
			if sqlKeywords[upper] {
				return nil, p.makeError(tagLine, fmt.Sprintf("'<%s>' is a reserved SQL keyword and cannot be used as a query identifier", tagName))
			}
			p.advance()

			paramRef := ""
			if p.cur.Type == PARAM_REF {
				paramRef = p.cur.Value
				p.advance()
			}

			body, err := p.parseBody(false)
			if err != nil {
				return nil, err
			}

			if p.cur.Type != TAG_CLOSE {
				return nil, p.makeError(p.cur.Line, fmt.Sprintf("expected '</>' to close query tag '<%s>'", tagName))
			}
			p.advance()

			qf.Queries = append(qf.Queries, QueryNode{
				Name:     tagName,
				ParamRef: paramRef,
				Body:     body,
			})

		case WHERE_TAG:
			return nil, p.makeError(p.cur.Line, "'<where>' is not allowed at namespace level")

		default:
			return nil, p.makeError(p.cur.Line, "unexpected token at namespace level")
		}
	}

	if p.cur.Type != TAG_CLOSE {
		return nil, p.makeError(p.cur.Line, "expected '</>' to close namespace tag")
	}
	p.advance()

	if err := p.checkLexErr(); err != nil {
		return nil, err
	}

	return qf, nil
}

func (p *Parser) parseBody(inFor bool) ([]SQLNode, error) {
	var nodes []SQLNode
	for {
		switch p.cur.Type {
		case SQL_TEXT:
			nodes = append(nodes, &SQLText{Text: p.cur.Value})
			p.advance()
		case BIND:
			nodes = append(nodes, &BindParam{Path: p.cur.Value})
			p.advance()
		case RAW:
			if !inFor {
				return nil, p.makeError(p.cur.Line, "'${}' is only allowed inside a 'for' statement")
			}
			nodes = append(nodes, &RawParam{Path: p.cur.Value})
			p.advance()
		case ESCAPE:
			nodes = append(nodes, &SQLText{Text: p.cur.Value})
			p.advance()
		case WHERE_TAG:
			wn, err := p.parseWhereNode(inFor)
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, wn)
		case EXPR_OPEN:
			p.advance()
			node, err := p.parseControlNode(inFor)
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, node)
		case TAG_CLOSE:
			return nodes, nil
		case EOF:
			return nil, p.makeError(p.cur.Line, "unexpected EOF in body")
		default:
			return nil, p.makeError(p.cur.Line, "unexpected token in body")
		}
	}
}

func (p *Parser) parseWhereNode(inFor bool) (*WhereNode, error) {
	p.advance() // consume WHERE_TAG
	children, err := p.parseBody(inFor)
	if err != nil {
		return nil, err
	}
	if p.cur.Type != TAG_CLOSE {
		return nil, p.makeError(p.cur.Line, "expected '</>' to close <where>")
	}
	p.advance()
	return &WhereNode{Children: children}, nil
}

// parseBlock parses body tokens between :{ and }.
// initText is the text extracted from the header SQL_TEXT token after :{.
// Returns when BLOCK_CLOSE is consumed.
func (p *Parser) parseBlock(initText string, inFor bool) ([]SQLNode, error) {
	var nodes []SQLNode

	if initText != "" {
		nodes = append(nodes, &SQLText{Text: initText})
	}

	for {
		switch p.cur.Type {
		case SQL_TEXT:
			nodes = append(nodes, &SQLText{Text: p.cur.Value})
			p.advance()
		case BIND:
			nodes = append(nodes, &BindParam{Path: p.cur.Value})
			p.advance()
		case RAW:
			if !inFor {
				return nil, p.makeError(p.cur.Line, "'${}' is only allowed inside a 'for' statement")
			}
			nodes = append(nodes, &RawParam{Path: p.cur.Value})
			p.advance()
		case ESCAPE:
			nodes = append(nodes, &SQLText{Text: p.cur.Value})
			p.advance()
		case WHERE_TAG:
			wn, err := p.parseWhereNode(inFor)
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, wn)
		case EXPR_OPEN:
			p.advance()
			node, err := p.parseControlNode(inFor)
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, node)
		case BLOCK_CLOSE:
			p.advance()
			return nodes, nil
		case EXPR_CLOSE, TAG_CLOSE, EOF:
			return nil, p.makeError(p.cur.Line, "unexpected end of control block")
		default:
			return nil, p.makeError(p.cur.Line, "unexpected token in control block")
		}
	}
}

// readControlHeader accumulates SQL_TEXT and ESCAPE tokens until ":{" is found.
// Returns: trimmed header text, text after ":{", starting line, error.
// Needed because ESCAPE tokens (e.g. != → <>) can split a control expression across tokens.
func (p *Parser) readControlHeader() (header, rest string, line int, err error) {
	if p.cur.Type != SQL_TEXT && p.cur.Type != ESCAPE {
		return "", "", p.cur.Line, p.makeError(p.cur.Line, "expected expression after '[['")
	}
	line = p.cur.Line
	var buf strings.Builder
	for {
		switch p.cur.Type {
		case SQL_TEXT:
			idx := strings.Index(p.cur.Value, ":{")
			if idx >= 0 {
				buf.WriteString(p.cur.Value[:idx])
				rest = p.cur.Value[idx+2:]
				p.advance()
				return strings.TrimSpace(buf.String()), rest, line, nil
			}
			buf.WriteString(p.cur.Value)
			p.advance()
		case ESCAPE:
			buf.WriteString(p.cur.Value)
			p.advance()
		default:
			return "", "", line, p.makeError(p.cur.Line,
				fmt.Sprintf("invalid expression in '[[ ]]': missing ':{' (got: %s)", strings.TrimSpace(buf.String())))
		}
	}
}

// parseControlNode parses the content of [[ ... ]] after EXPR_OPEN is consumed.
func (p *Parser) parseControlNode(inFor bool) (SQLNode, error) {
	header, rest, headerLine, err := p.readControlHeader()
	if err != nil {
		return nil, err
	}

	firstWord := strings.SplitN(header, " ", 2)[0]

	var node SQLNode
	switch firstWord {
	case "if":
		node, err = p.parseIfNode(header, headerLine, rest, inFor)
	case "switch":
		node, err = p.parseSwitchNode(header, headerLine, rest, inFor)
	case "for":
		node, err = p.parseForNode(header, headerLine, rest)
	default:
		return nil, p.makeError(headerLine, fmt.Sprintf("invalid expression in '[[ ]]': %s", header))
	}
	if err != nil {
		return nil, err
	}

	if p.cur.Type != EXPR_CLOSE {
		return nil, p.makeError(p.cur.Line, "expected ']]' to close control statement")
	}
	p.advance()

	return node, nil
}

func (p *Parser) parseIfNode(header string, line int, rest string, inFor bool) (*IfNode, error) {
	cond := strings.TrimSpace(strings.TrimPrefix(header, "if"))
	if cond == "" {
		return nil, p.makeError(line, "missing condition in 'if' statement")
	}

	thenNodes, err := p.parseBlock(rest, inFor)
	if err != nil {
		return nil, err
	}

	node := &IfNode{Cond: cond, Then: thenNodes}

	for p.cur.Type == SQL_TEXT {
		trimmed := strings.TrimSpace(p.cur.Value)
		if !strings.HasPrefix(trimmed, "else") {
			break
		}

		// Use readControlHeader to accumulate across any ESCAPE tokens in the condition.
		elseHeader, rest2, _, err := p.readControlHeader()
		if err != nil {
			return nil, err
		}

		if strings.HasPrefix(elseHeader, "else if") {
			elseifCond := strings.TrimSpace(strings.TrimPrefix(elseHeader, "else if"))
			body, err := p.parseBlock(rest2, inFor)
			if err != nil {
				return nil, err
			}
			node.ElseIfs = append(node.ElseIfs, ElseIfClause{Cond: elseifCond, Body: body})
		} else {
			// plain else
			body, err := p.parseBlock(rest2, inFor)
			if err != nil {
				return nil, err
			}
			node.Else = body
			break
		}
	}

	return node, nil
}

func (p *Parser) parseSwitchNode(header string, line int, rest string, inFor bool) (*SwitchNode, error) {
	// header: "switch (expr)"
	expr := ""
	trimmed := strings.TrimPrefix(header, "switch")
	trimmed = strings.TrimSpace(trimmed)
	if strings.HasPrefix(trimmed, "(") && strings.HasSuffix(trimmed, ")") {
		expr = trimmed[1 : len(trimmed)-1]
	} else {
		expr = trimmed
	}

	cases, defaultNodes, err := p.parseSwitchBody(rest, line, inFor)
	if err != nil {
		return nil, err
	}

	return &SwitchNode{Expr: expr, Cases: cases, Default: defaultNodes}, nil
}

func (p *Parser) parseSwitchBody(initText string, startLine int, inFor bool) ([]CaseClause, []SQLNode, error) {
	var cases []CaseClause
	var defaultNodes []SQLNode
	currentText := initText

	for {
		trimmed := strings.TrimSpace(currentText)

		if strings.HasPrefix(trimmed, "case ") || strings.HasPrefix(trimmed, "case\t") {
			colonIdx := strings.Index(currentText, ":{")
			if colonIdx < 0 {
				return nil, nil, p.makeError(startLine, "expected ':{' after 'case' declaration")
			}
			caseHeader := strings.TrimSpace(currentText[:colonIdx])
			bodyInitText := currentText[colonIdx+2:]

			caseValue := extractCaseValue(caseHeader)

			body, err := p.parseBlock(bodyInitText, inFor)
			if err != nil {
				return nil, nil, err
			}
			cases = append(cases, CaseClause{Value: caseValue, Body: body})

		} else if strings.HasPrefix(trimmed, "default") {
			colonIdx := strings.Index(currentText, ":{")
			if colonIdx < 0 {
				return nil, nil, p.makeError(startLine, "expected ':{' after 'default'")
			}
			bodyInitText := currentText[colonIdx+2:]

			body, err := p.parseBlock(bodyInitText, inFor)
			if err != nil {
				return nil, nil, err
			}
			defaultNodes = body

		} else if trimmed != "" {
			return nil, nil, p.makeError(startLine, fmt.Sprintf("unexpected content in switch body: %s", trimmed))
		}

		switch p.cur.Type {
		case BLOCK_CLOSE:
			p.advance()
			return cases, defaultNodes, nil
		case SQL_TEXT:
			currentText = p.cur.Value
			p.advance()
		case EXPR_CLOSE, TAG_CLOSE, EOF:
			return nil, nil, p.makeError(p.cur.Line, "unexpected end of switch body")
		default:
			return nil, nil, p.makeError(p.cur.Line, "unexpected token in switch body")
		}
	}
}

// extractCaseValue extracts the value from "case ("val")" → "val" (with quotes stripped).
func extractCaseValue(header string) string {
	// header: `case ("someValue")`
	open := strings.Index(header, "(")
	close := strings.LastIndex(header, ")")
	if open < 0 || close < 0 || close <= open {
		return header
	}
	inner := strings.TrimSpace(header[open+1 : close])
	// strip surrounding quotes if present
	if len(inner) >= 2 && inner[0] == '"' && inner[len(inner)-1] == '"' {
		return inner[1 : len(inner)-1]
	}
	return inner
}

func (p *Parser) parseForNode(header string, line int, rest string) (*ForNode, error) {
	// header: "for item in list" / "for item as column in list" / "for key, value in map"
	withoutFor := strings.TrimSpace(strings.TrimPrefix(header, "for"))

	inIdx := strings.LastIndex(withoutFor, " in ")
	if inIdx < 0 {
		return nil, p.makeError(line, fmt.Sprintf("invalid 'for' syntax: %s", header))
	}
	decl := strings.TrimSpace(withoutFor[:inIdx])
	collection := strings.TrimSpace(withoutFor[inIdx+4:])

	node := &ForNode{Collection: collection}

	if strings.Contains(decl, ",") {
		return nil, p.makeError(line, "'for map' syntax is removed in v2.1. Use '[[for item in collection:{...}]]' with anonymous struct slice instead.")
	}

	// scalar / struct pattern
	node.ItemVar = decl

	body, err := p.parseBlock(rest, true)
	if err != nil {
		return nil, err
	}
	node.Body = body

	return node, nil
}

