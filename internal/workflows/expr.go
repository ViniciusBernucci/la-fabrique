package workflows

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Expressões das condições: linguagem pequena e determinística, sem chamadas de
// função nem efeitos. Gramática:
//
//	or      = and { "||" and }
//	and     = cmp { "&&" cmp }
//	cmp     = unary [ ("==" | "!=" | "<" | "<=" | ">" | ">=") unary ]
//	unary   = "!" unary | literal | caminho
//
// Literais: true, false, null, números e textos entre aspas simples ou duplas.
// Caminhos (a.b.c) leem variáveis; variável ausente vale null.

// ErrExpr indica expressão inválida ou impossível de avaliar.
var ErrExpr = errors.New("expressão inválida")

type exprNode interface {
	eval(vars map[string]any) (any, error)
}

// Expr é uma expressão já analisada.
type Expr struct{ root exprNode }

// ParseExpr analisa o texto. Erros dizem a posição aproximada.
func ParseExpr(src string) (*Expr, error) {
	if strings.TrimSpace(src) == "" || len(src) > 500 {
		return nil, fmt.Errorf("%w: vazia ou longa demais", ErrExpr)
	}
	p := &exprParser{src: src}
	n, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	p.skip()
	if p.pos != len(p.src) {
		return nil, fmt.Errorf("%w: texto inesperado na posição %d", ErrExpr, p.pos+1)
	}
	return &Expr{root: n}, nil
}

// Eval avalia como booleano (verdade conforme as regras abaixo).
func (e *Expr) Eval(vars map[string]any) (bool, error) {
	v, err := e.root.eval(vars)
	if err != nil {
		return false, err
	}
	return truthy(v), nil
}

type exprParser struct {
	src string
	pos int
}

func (p *exprParser) skip() {
	for p.pos < len(p.src) && (p.src[p.pos] == ' ' || p.src[p.pos] == '\t') {
		p.pos++
	}
}

func (p *exprParser) peek(s string) bool {
	p.skip()
	return strings.HasPrefix(p.src[p.pos:], s)
}

func (p *exprParser) take(s string) bool {
	if p.peek(s) {
		p.pos += len(s)
		return true
	}
	return false
}

func (p *exprParser) parseOr() (exprNode, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.take("||") {
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = logic{op: "||", l: left, r: right}
	}
	return left, nil
}

func (p *exprParser) parseAnd() (exprNode, error) {
	left, err := p.parseCmp()
	if err != nil {
		return nil, err
	}
	for p.take("&&") {
		right, err := p.parseCmp()
		if err != nil {
			return nil, err
		}
		left = logic{op: "&&", l: left, r: right}
	}
	return left, nil
}

func (p *exprParser) parseCmp() (exprNode, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for _, op := range []string{"==", "!=", "<=", ">=", "<", ">"} {
		if p.take(op) {
			right, err := p.parseUnary()
			if err != nil {
				return nil, err
			}
			return cmp{op: op, l: left, r: right}, nil
		}
	}
	return left, nil
}

func (p *exprParser) parseUnary() (exprNode, error) {
	p.skip()
	if p.pos >= len(p.src) {
		return nil, fmt.Errorf("%w: operando esperado no fim", ErrExpr)
	}
	if p.peek("!") && !p.peek("!=") {
		p.pos++
		inner, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return not{inner}, nil
	}
	c := p.src[p.pos]
	switch {
	case c == '\'' || c == '"':
		end := strings.IndexByte(p.src[p.pos+1:], c)
		if end < 0 {
			return nil, fmt.Errorf("%w: texto sem fechar aspas", ErrExpr)
		}
		s := p.src[p.pos+1 : p.pos+1+end]
		p.pos += end + 2
		return lit{s}, nil
	case c >= '0' && c <= '9' || c == '-':
		start := p.pos
		p.pos++
		for p.pos < len(p.src) && (p.src[p.pos] >= '0' && p.src[p.pos] <= '9' || p.src[p.pos] == '.') {
			p.pos++
		}
		f, err := strconv.ParseFloat(p.src[start:p.pos], 64)
		if err != nil {
			return nil, fmt.Errorf("%w: número inválido na posição %d", ErrExpr, start+1)
		}
		return lit{f}, nil
	case isIdentStart(c):
		start := p.pos
		for p.pos < len(p.src) && (isIdentStart(p.src[p.pos]) || p.src[p.pos] >= '0' && p.src[p.pos] <= '9' || p.src[p.pos] == '.') {
			p.pos++
		}
		word := p.src[start:p.pos]
		switch word {
		case "true":
			return lit{true}, nil
		case "false":
			return lit{false}, nil
		case "null":
			return lit{nil}, nil
		}
		if strings.HasSuffix(word, ".") || strings.Contains(word, "..") {
			return nil, fmt.Errorf("%w: caminho inválido na posição %d", ErrExpr, start+1)
		}
		return path(strings.Split(word, ".")), nil
	}
	return nil, fmt.Errorf("%w: caractere inesperado na posição %d", ErrExpr, p.pos+1)
}

func isIdentStart(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

type lit struct{ v any }

func (l lit) eval(map[string]any) (any, error) { return l.v, nil }

type path []string

func (p path) eval(vars map[string]any) (any, error) {
	var cur any = vars
	for _, k := range p {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, nil
		}
		cur = m[k]
	}
	return cur, nil
}

type not struct{ inner exprNode }

func (n not) eval(vars map[string]any) (any, error) {
	v, err := n.inner.eval(vars)
	return !truthy(v), err
}

type logic struct {
	op   string
	l, r exprNode
}

func (l logic) eval(vars map[string]any) (any, error) {
	a, err := l.l.eval(vars)
	if err != nil {
		return nil, err
	}
	if l.op == "&&" && !truthy(a) {
		return false, nil
	}
	if l.op == "||" && truthy(a) {
		return true, nil
	}
	b, err := l.r.eval(vars)
	return truthy(b), err
}

type cmp struct {
	op   string
	l, r exprNode
}

func toNum(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	}
	return 0, false
}

func (c cmp) eval(vars map[string]any) (any, error) {
	a, err := c.l.eval(vars)
	if err != nil {
		return nil, err
	}
	b, err := c.r.eval(vars)
	if err != nil {
		return nil, err
	}
	switch c.op {
	case "==":
		return equal(a, b), nil
	case "!=":
		return !equal(a, b), nil
	}
	x, okx := toNum(a)
	y, oky := toNum(b)
	if !okx || !oky {
		return nil, fmt.Errorf("%w: %q exige números", ErrExpr, c.op)
	}
	switch c.op {
	case "<":
		return x < y, nil
	case "<=":
		return x <= y, nil
	case ">":
		return x > y, nil
	default:
		return x >= y, nil
	}
}

func equal(a, b any) bool {
	if x, ok := toNum(a); ok {
		y, ok2 := toNum(b)
		return ok2 && x == y
	}
	return a == b
}

func truthy(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case string:
		return x != ""
	}
	if n, ok := toNum(v); ok {
		return n != 0
	}
	return true
}
