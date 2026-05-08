package parser

import (
	"fmt"

	"github.com/mcabezas/archlang/internal/ast"
	"github.com/mcabezas/archlang/internal/lexer"
	"github.com/mcabezas/archlang/internal/token"
)

type Parser struct {
	l         *lexer.Lexer
	errors    []string
	curToken  token.Token
	peekToken token.Token
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{l: l, errors: []string{}}
	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) Parse() *ast.Architecture {
	arch := &ast.Architecture{}
	arch.Statements = []ast.Statement{}

	for !p.curTokenIs(token.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			arch.Statements = append(arch.Statements, stmt)
		}
		p.nextToken()
	}

	return arch
}

func (p *Parser) parseStatement() ast.Statement {
	switch p.curToken.Type {
	case token.COMPONENT:
		return p.parseComponentStatement()
	case token.SERVICE:
		return p.parseServiceStatement()
	case token.COLLABORATION:
		return p.parseCollaborationStatement()
	default:
		p.addError("unexpected token %q at line %d, column %d",
			p.curToken.Literal, p.curToken.Line, p.curToken.Column)
		return nil
	}
}

func (p *Parser) parseComponentStatement() *ast.ComponentStatement {
	stmt := &ast.ComponentStatement{Token: p.curToken}

	if !p.expectPeek(token.IDENT) {
		return nil
	}

	stmt.Name = p.curToken.Literal
	return stmt
}

func (p *Parser) parseServiceStatement() *ast.ServiceStatement {
	stmt := &ast.ServiceStatement{Token: p.curToken}

	if !p.expectPeek(token.IDENT) {
		return nil
	}

	stmt.Name = p.curToken.Literal
	return stmt
}

func (p *Parser) parseCollaborationStatement() *ast.CollaborationStatement {
	stmt := &ast.CollaborationStatement{Token: p.curToken}

	if !p.expectPeek(token.IDENT) {
		return nil
	}
	stmt.Source = p.curToken.Literal

	if !p.expectPeek(token.ARROW) {
		return nil
	}

	if !p.expectPeek(token.IDENT) {
		return nil
	}
	stmt.Target = p.curToken.Literal

	return stmt
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) curTokenIs(t token.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t token.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) expectPeek(t token.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.peekError(t)
	return false
}

func (p *Parser) peekError(t token.TokenType) {
	p.addError("expected %s, got %s instead at line %d, column %d",
		t, p.peekToken.Type, p.peekToken.Line, p.peekToken.Column)
}

func (p *Parser) addError(format string, args ...interface{}) {
	p.errors = append(p.errors, fmt.Sprintf(format, args...))
}
