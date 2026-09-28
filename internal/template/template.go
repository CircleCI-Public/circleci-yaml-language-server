// Package template finds the errors orb-service reports when it renders a
// config value as a template: a `<< >>` tag, a `<<# >>` or `<<^ >>` section,
// or the expression inside a tag. It follows orb-service's replacer, in
// server/src-java/com/circleci/replacer.
package template

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/CircleCI-Public/expr/parsers"
	"github.com/CircleCI-Public/expr/scanners"
	"github.com/CircleCI-Public/expr/tokens"
)

// maxExpressionLength is in characters, as orb-service counts them.
const maxExpressionLength = 2048

// Problem is what's wrong with a template, and where, as byte offsets into
// it.
type Problem struct {
	Start, End int
	Message    string
}

// Check returns the first problem in text, as orb-service stops at the first.
func Check(text string) (Problem, bool) {
	p := parser{tokens: scan(text), text: text}
	err := p.parse()
	if err == nil {
		return Problem{}, false
	}
	var problem Problem
	if !errors.As(err, &problem) {
		return Problem{}, false
	}
	return problem, true
}

func (p Problem) Error() string { return p.Message }

// Reference is a name a template uses, such as parameters.version, and where,
// as byte offsets into the template.
type Reference struct {
	Name       string
	Start, End int
}

// References returns the names used in text's tags and sections, up to its
// first problem.
func References(text string) []Reference {
	p := parser{tokens: scan(text), text: text}
	_ = p.parse()
	return p.references
}

type tokenType int

const (
	stringToken tokenType = iota
	tagStart
	sectionStart
	invertedSectionStart
	sectionEnd
	tagEnd
	eof
)

type token struct {
	kind  tokenType
	text  string
	start int
}

func (t token) end() int { return t.start + len(t.text) }

// scan splits text as orb-service's replacer scanner does. `\<<` is text,
// and `\\<<` is a `\` followed by a tag.
func scan(text string) []token {
	toks := []token{}
	add := func(kind tokenType, start, end int) {
		toks = append(toks, token{kind: kind, text: text[start:end], start: start})
	}
	tag := func(i int) int {
		if i+2 < len(text) {
			switch text[i+2] {
			case '#':
				add(sectionStart, i, i+3)
				return i + 3
			case '^':
				add(invertedSectionStart, i, i+3)
				return i + 3
			case '/':
				add(sectionEnd, i, i+3)
				return i + 3
			}
		}
		add(tagStart, i, i+2)
		return i + 2
	}
	atText := func(i int) bool {
		rest := text[i:]
		return !strings.HasPrefix(rest, "<<") && !strings.HasPrefix(rest, ">>") &&
			!strings.HasPrefix(rest, `\<<`) && !strings.HasPrefix(rest, `\\<<`)
	}

	for i := 0; i < len(text); {
		rest := text[i:]
		switch {
		case strings.HasPrefix(rest, "<<"):
			i = tag(i)
		case strings.HasPrefix(rest, ">>"):
			add(tagEnd, i, i+2)
			i += 2
		case strings.HasPrefix(rest, `\<<`):
			add(stringToken, i, i+3)
			i += 3
		case strings.HasPrefix(rest, `\\<<`):
			add(stringToken, i, i+1)
			i = tag(i + 2)
		default:
			start := i
			for i < len(text) && atText(i) {
				i++
			}
			add(stringToken, start, i)
		}
	}
	toks = append(toks, token{kind: eof, start: len(text)})
	return toks
}

// identifierPattern is how lax orb-service is about a section's name, and
// about a tag that isn't an expression.
var identifierPattern = regexp.MustCompile(`^[\w-]+(\.\S+)*$`)

type parser struct {
	tokens     []token
	current    int
	text       string
	references []Reference
}

func (p *parser) parse() error {
	for !p.check(eof) {
		if err := p.block(); err != nil {
			return err
		}
	}
	return nil
}

func (p *parser) block() error {
	switch {
	case p.match(sectionStart, invertedSectionStart):
		return p.section()
	case p.match(tagStart):
		return p.tag()
	case p.match(sectionEnd):
		return problemAt(p.previous(), "Expected section end tag to match start tag")
	}
	return p.textBlock()
}

func (p *parser) section() error {
	op := p.previous()
	name, err := p.identifier()
	if err != nil {
		return err
	}
	if _, err := p.consume(tagEnd, p.peek(), "Unclosed '<<' tag"); err != nil {
		return err
	}

	for !p.check(eof) {
		if p.check(sectionEnd) && p.current+1 < len(p.tokens) && strings.TrimSpace(p.tokens[p.current+1].text) == name {
			break
		}
		if err := p.block(); err != nil {
			return err
		}
	}

	if _, err := p.consume(sectionEnd, op, "Unclosed section, expected '<</'"); err != nil {
		return err
	}
	endName, err := p.identifier()
	if err != nil {
		return err
	}
	if endName != name {
		return problemAt(p.previous(), "Expected section end tag to match start tag")
	}
	_, err = p.consume(tagEnd, p.peek(), "Unclosed '<<' tag")
	return err
}

func (p *parser) identifier() (string, error) {
	tok, err := p.consume(stringToken, p.peek(), "Expected identifier")
	if err != nil {
		return "", err
	}
	name := strings.TrimSpace(tok.text)
	if !identifierPattern.MatchString(name) {
		return "", problemAt(tok, "Expected valid identifier")
	}
	p.references = append(p.references, wholeReference(tok))
	return name, nil
}

func (p *parser) tag() error {
	start := p.previous()
	body, err := p.consume(stringToken, start, "Expected expression after '<<'")
	if err != nil {
		return err
	}
	if n := utf8.RuneCountInString(body.text); n > maxExpressionLength {
		return problemAt(body, fmt.Sprintf("Expressions must be less than %d characters, this one is %d", maxExpressionLength, n))
	}
	if _, err := p.consume(tagEnd, start, "Unclosed '<<' tag ('<<' must be escaped as '\\<<')"); err != nil {
		return err
	}
	references, err := parseExpression(body)
	p.references = append(p.references, references...)
	return err
}

// wholeReference is the name that's all of tok but the space around it.
func wholeReference(tok token) Reference {
	start := tok.start + len(tok.text) - len(strings.TrimLeftFunc(tok.text, unicode.IsSpace))
	name := strings.TrimSpace(tok.text)
	return Reference{Name: name, Start: start, End: start + len(name)}
}

// parseExpression parses a tag's body as an expression, and returns the names
// it uses. A body that uses a character expressions don't allow is taken
// whole as a name, which is how orb-service still accepts pipeline
// parameters named like URLs.
func parseExpression(body token) ([]Reference, error) {
	toks, err := scanners.New(body.text).Scan()
	if err != nil {
		var scanErr scanners.Error
		if !errors.As(err, &scanErr) {
			return nil, problemAt(body, "Invalid expression")
		}
		if scanErr.Type == scanners.UNEXPECTED_CHARACTER && identifierPattern.MatchString(strings.TrimSpace(body.text)) {
			return []Reference{wholeReference(body)}, nil
		}
		start := body.start + byteOffset(body.text, scanErr.Pos)
		_, size := utf8.DecodeRuneInString(body.text[start-body.start:])
		return nil, Problem{Start: start, End: start + size, Message: firstLine(scanErr.AsErrorMessage(body.text))}
	}

	expression, err := parsers.New(toks).Parse()
	if err == nil {
		references := []Reference{}
		for _, name := range identifiers(expression) {
			start := body.start + byteOffset(body.text, name.CharPos)
			references = append(references, Reference{Name: name.Lexeme, Start: start, End: start + len(name.Lexeme)})
		}
		return references, nil
	}
	var parseErr parsers.Error
	if !errors.As(err, &parseErr) {
		return nil, problemAt(body, "Invalid expression")
	}
	start := body.start + byteOffset(body.text, parseErr.Token.CharPos)
	end := start + len(parseErr.Token.Lexeme)
	if parseErr.Token.Lexeme == "" {
		// The expression ended too soon, so point at all of it.
		start, end = body.start, body.end()
	}
	return nil, Problem{Start: start, End: end, Message: firstLine(parseErr.AsErrorMessage(body.text))}
}

// identifiers are the names an expression uses, in order.
func identifiers(expression parsers.Expr) []tokens.Token {
	switch e := expression.(type) {
	case parsers.Identifier:
		return []tokens.Token{e.Name}
	case parsers.Logical:
		return append(identifiers(e.Left), identifiers(e.Right)...)
	case parsers.Binary:
		return append(identifiers(e.Left), identifiers(e.Right)...)
	case parsers.Infix:
		return append(identifiers(e.Left), identifiers(e.Right)...)
	case parsers.Unary:
		return identifiers(e.Right)
	case parsers.Grouping:
		return identifiers(e.Expression)
	}
	return nil
}

// byteOffset converts expr's position, in characters, to one in bytes.
func byteOffset(s string, chars int) int {
	for i := range s {
		if chars == 0 {
			return i
		}
		chars--
	}
	return len(s)
}

// firstLine is the explanation from one of expr's messages, without the
// lines that quote the expression and point into it.
func firstLine(message string) string {
	line, _, _ := strings.Cut(message, "\n")
	return strings.TrimSuffix(line, ":")
}

func problemAt(tok token, message string) Problem {
	return Problem{Start: tok.start, End: tok.end(), Message: message}
}

func (p *parser) textBlock() error {
	start := p.current
	for p.check(stringToken) || p.check(tagEnd) {
		p.current++
	}
	if p.current == start {
		return problemAt(p.peek(), "Expected string")
	}
	return nil
}

func (p *parser) match(kinds ...tokenType) bool {
	for _, kind := range kinds {
		if p.check(kind) {
			p.current++
			return true
		}
	}
	return false
}

func (p *parser) consume(kind tokenType, at token, message string) (token, error) {
	if p.check(kind) {
		p.current++
		return p.previous(), nil
	}
	return token{}, problemAt(at, message)
}

func (p *parser) check(kind tokenType) bool { return p.peek().kind == kind }

func (p *parser) peek() token { return p.tokens[p.current] }

func (p *parser) previous() token { return p.tokens[p.current-1] }
