// Package template parses a config value as a template: `<< >>` tags,
// `<<# >>` and `<<^ >>` sections, and the expressions inside tags. It finds
// the errors the reference compiler reports when it renders one, and follows
// that compiler's replacer.
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

// maxExpressionLength is in characters, as the reference compiler counts
// them.
const maxExpressionLength = 2048

// Problem is what's wrong with a template, and where, as byte offsets into
// it.
type Problem struct {
	Start, End int
	Message    string
}

// Template is a parsed template's blocks, in order.
type Template []Block

// Block is a Text, a Tag or a Section.
type Block interface{ isBlock() }

// Text is literal text, with its escapes as written.
type Text struct {
	Value string
}

// Tag is a `<< >>` tag, with Start and End as byte offsets into the
// template. Name is set when the body is a bare name. Expr is nil when the
// body is a name expressions don't allow, such as one shaped like a URL.
type Tag struct {
	Expr       parsers.Expr
	Name       string
	Start, End int
}

// Section is a `<<# >>` section, or a `<<^ >>` one when Inverted, from its
// start tag to the end of its end tag.
type Section struct {
	Name       string
	Inverted   bool
	Body       Template
	Start, End int
}

func (Text) isBlock()    {}
func (Tag) isBlock()     {}
func (Section) isBlock() {}

// Parse parses text, stopping at its first problem.
func Parse(text string) (Template, *Problem) {
	p := parser{tokens: scan(text), text: text, tree: true}
	t, err := p.parse()
	if err == nil {
		return t, nil
	}
	var problem Problem
	if !errors.As(err, &problem) {
		problem = Problem{Message: err.Error()}
	}
	return nil, &problem
}

// Check returns the first problem in text, as the reference compiler stops
// at the first.
func Check(text string) (Problem, bool) {
	p := parser{tokens: scan(text), text: text}
	_, err := p.parse()
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

// IsOnlyTag reports whether text is a single `<< >>` tag and nothing else,
// such as `<< pipeline.number >>` or `<< pipeline.git.tag and 1 or 0 >>`. The
// compiler gives such a value the type of what's inside it, rather than
// making it a string.
func IsOnlyTag(text string) bool {
	toks := scan(text)
	if len(toks) != 4 || toks[0].kind != tagStart || toks[1].kind != stringToken || toks[2].kind != tagEnd {
		return false
	}
	_, found := Check(text)
	return !found
}

// References returns the names used in text's tags and sections, up to its
// first problem.
func References(text string) []Reference {
	p := parser{tokens: scan(text), text: text}
	_, _ = p.parse()
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

// scan splits text as the reference compiler's replacer scanner does. `\<<`
// is text, and `\\<<` is a `\` followed by a tag.
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

// identifierPattern is how lax the reference compiler is about a section's
// name, and about a tag that isn't an expression.
var identifierPattern = regexp.MustCompile(`^[\w-]+(\.\S+)*$`)

type parser struct {
	tokens     []token
	current    int
	text       string
	references []Reference
	// tree is whether to build the blocks, which validating alone doesn't
	// need.
	tree bool
}

func (p *parser) parse() (Template, error) {
	return p.blocks(func() bool { return false })
}

// blocks parses blocks until the end of the text, or until done.
func (p *parser) blocks(done func() bool) (Template, error) {
	var t Template
	for !p.check(eof) && !done() {
		b, err := p.block()
		if err != nil {
			return nil, err
		}
		if p.tree {
			t = append(t, b)
		}
	}
	return t, nil
}

func (p *parser) block() (Block, error) {
	switch {
	case p.match(sectionStart, invertedSectionStart):
		return p.section()
	case p.match(tagStart):
		return p.tag()
	case p.match(sectionEnd):
		return nil, problemAt(p.previous(), "Expected section end tag to match start tag")
	}
	return p.textBlock()
}

func (p *parser) section() (Block, error) {
	op := p.previous()
	name, err := p.identifier()
	if err != nil {
		return nil, err
	}
	if _, err := p.consume(tagEnd, p.peek(), "Unclosed '<<' tag"); err != nil {
		return nil, err
	}

	body, err := p.blocks(func() bool {
		return p.check(sectionEnd) && p.current+1 < len(p.tokens) && strings.TrimSpace(p.tokens[p.current+1].text) == name
	})
	if err != nil {
		return nil, err
	}

	if _, err := p.consume(sectionEnd, op, "Unclosed section, expected '<</'"); err != nil {
		return nil, err
	}
	endName, err := p.identifier()
	if err != nil {
		return nil, err
	}
	if endName != name {
		return nil, problemAt(p.previous(), "Expected section end tag to match start tag")
	}
	end, err := p.consume(tagEnd, p.peek(), "Unclosed '<<' tag")
	if err != nil || !p.tree {
		return nil, err
	}
	return Section{Name: name, Inverted: op.kind == invertedSectionStart, Body: body, Start: op.start, End: end.end()}, nil
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

func (p *parser) tag() (Block, error) {
	start := p.previous()
	body, err := p.consume(stringToken, start, "Expected expression after '<<'")
	if err != nil {
		return nil, err
	}
	if n := utf8.RuneCountInString(body.text); n > maxExpressionLength {
		return nil, problemAt(body, fmt.Sprintf("Expressions must be less than %d characters, this one is %d", maxExpressionLength, n))
	}
	end, err := p.consume(tagEnd, start, "Unclosed '<<' tag ('<<' must be escaped as '\\<<')")
	if err != nil {
		return nil, err
	}
	expression, references, err := parseExpression(body, true)
	p.references = append(p.references, references...)
	if err != nil || !p.tree {
		return nil, err
	}
	tag := Tag{Expr: expression, Start: start.start, End: end.end()}
	switch e := expression.(type) {
	case nil:
		tag.Name = strings.TrimSpace(body.text)
	case parsers.Identifier:
		tag.Name = e.Name.Lexeme
	}
	return tag, nil
}

// wholeReference is the name that's all of tok but the space around it.
func wholeReference(tok token) Reference {
	start := tok.start + len(tok.text) - len(strings.TrimLeftFunc(tok.text, unicode.IsSpace))
	name := strings.TrimSpace(tok.text)
	return Reference{Name: name, Start: start, End: start + len(name)}
}

// Expression parses text as an expression on its own, as a workflow's
// `when: pipeline.git.branch == "main"` is, and returns the names it uses.
func Expression(text string) ([]Reference, *Problem) {
	body := token{kind: stringToken, text: text}
	if n := utf8.RuneCountInString(text); n > maxExpressionLength {
		problem := problemAt(body, fmt.Sprintf("Expressions must be less than %d characters, this one is %d", maxExpressionLength, n))
		return nil, &problem
	}
	_, references, err := parseExpression(body, false)
	var problem Problem
	if errors.As(err, &problem) {
		return nil, &problem
	}
	return references, nil
}

// parseExpression parses a tag's body as an expression, and returns it with
// the names it uses. In a tag, a body that uses a character expressions don't
// allow is taken whole as a name, with no expression, which is how the
// reference compiler still accepts pipeline parameters named like URLs.
func parseExpression(body token, inTag bool) (parsers.Expr, []Reference, error) {
	toks, err := scanners.New(body.text).Scan()
	if err != nil {
		var scanErr scanners.Error
		if !errors.As(err, &scanErr) {
			return nil, nil, problemAt(body, "Invalid expression")
		}
		if inTag && scanErr.Type == scanners.UNEXPECTED_CHARACTER && identifierPattern.MatchString(strings.TrimSpace(body.text)) {
			return nil, []Reference{wholeReference(body)}, nil
		}
		start := body.start + byteOffset(body.text, scanErr.Pos)
		_, size := utf8.DecodeRuneInString(body.text[start-body.start:])
		return nil, nil, Problem{Start: start, End: start + size, Message: firstLine(scanErr.AsErrorMessage(body.text))}
	}

	expression, err := parsers.New(toks).Parse()
	if err == nil {
		references := []Reference{}
		for _, name := range identifiers(expression) {
			start := body.start + byteOffset(body.text, name.CharPos)
			references = append(references, Reference{Name: name.Lexeme, Start: start, End: start + len(name.Lexeme)})
		}
		return expression, references, nil
	}
	var parseErr parsers.Error
	if !errors.As(err, &parseErr) {
		return nil, nil, problemAt(body, "Invalid expression")
	}
	start := body.start + byteOffset(body.text, parseErr.Token.CharPos)
	end := start + len(parseErr.Token.Lexeme)
	message := firstLine(parseErr.AsErrorMessage(body.text))
	if parseErr.Token.Lexeme == "" {
		// The expression ended too soon, so point at all of it.
		start, end = body.start, body.end()
		message = strings.Replace(message, `found ""`, "found end of expression", 1)
	}
	return nil, nil, Problem{Start: start, End: end, Message: message}
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

func (p *parser) textBlock() (Block, error) {
	start := p.peek()
	for p.check(stringToken) || p.check(tagEnd) {
		p.current++
	}
	if p.peek() == start {
		return nil, problemAt(start, "Expected string")
	}
	if !p.tree {
		return nil, nil
	}
	return Text{Value: p.text[start.start:p.previous().end()]}, nil
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
