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
	case token.DOMAIN:
		return &ast.DomainStatement{Token: p.curToken}
	case token.IMPORT:
		return p.parseImportStatement()
	case token.COMPONENT:
		return p.parseComponentStatement()
	case token.SERVICE:
		return p.parseServiceStatement()
	case token.COLLABORATION:
		return p.parseCollaborationStatement()
	case token.IDENT:
		// name.attr = value (attribute assignment)
		if p.peekTokenIs(token.DOT) {
			return p.parseAttributeStatement()
		}
		p.addError("unexpected identifier %q at line %d, column %d",
			p.curToken.Literal, p.curToken.Line, p.curToken.Column)
		return nil
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

	if p.peekTokenIs(token.LBRACE) {
		p.nextToken() // consume {
		p.parseComponentAttributes(&stmt.Frontend, &stmt.Infra)
	}

	return stmt
}

func (p *Parser) parseServiceStatement() *ast.ServiceStatement {
	stmt := &ast.ServiceStatement{Token: p.curToken}

	if !p.expectPeek(token.IDENT) {
		return nil
	}

	stmt.Name = p.curToken.Literal

	if p.peekTokenIs(token.LBRACE) {
		p.nextToken() // consume {
		p.parseComponentAttributes(&stmt.Frontend, nil)
	}

	return stmt
}

func (p *Parser) parseComponentAttributes(frontend *bool, infra *string) {
	for !p.peekTokenIs(token.RBRACE) && !p.peekTokenIs(token.EOF) {
		p.nextToken()
		switch p.curToken.Type {
		case token.COMMA:
			continue
		case token.FRONTEND:
			*frontend = true
		case token.INFRA:
			if infra == nil {
				p.addError("infra type not allowed on services at line %d", p.curToken.Line)
				return
			}
			if !p.expectPeek(token.COLON) {
				return
			}
			if !p.expectPeek(token.IDENT) {
				return
			}
			*infra = p.curToken.Literal
		default:
			p.addError("unexpected token %q in attribute block at line %d", p.curToken.Literal, p.curToken.Line)
			return
		}
	}
	if !p.expectPeek(token.RBRACE) {
		return
	}
}

func (p *Parser) parseImportStatement() *ast.ImportStatement {
	stmt := &ast.ImportStatement{Token: p.curToken}

	if !p.expectPeek(token.IDENT) {
		return nil
	}
	stmt.Package = p.curToken.Literal
	stmt.Alias = p.curToken.Literal

	if p.peekTokenIs(token.AS) {
		p.nextToken()
		if !p.expectPeek(token.IDENT) {
			return nil
		}
		stmt.Alias = p.curToken.Literal
	}

	return stmt
}

func (p *Parser) parseCollaborationStatement() *ast.CollaborationStatement {
	stmt := &ast.CollaborationStatement{Token: p.curToken}

	if !p.expectPeek(token.IDENT) {
		return nil
	}
	stmt.Source = p.parseComponentRef()

	if !p.expectPeek(token.ARROW) {
		return nil
	}

	if !p.expectPeek(token.IDENT) {
		return nil
	}
	stmt.Target = p.parseComponentRef()

	return stmt
}

func (p *Parser) parseAttributeStatement() *ast.AttributeStatement {
	stmt := &ast.AttributeStatement{Token: p.curToken}
	stmt.Component = p.curToken.Literal

	p.nextToken() // consume dot

	// Next token is the attribute name
	p.nextToken()
	if p.curToken.Type == token.FRONTEND {
		stmt.Attribute = "frontend"
	} else if p.curToken.Type == token.IDENT {
		stmt.Attribute = p.curToken.Literal
	} else {
		p.addError("expected attribute name, got %s at line %d", p.curToken.Type, p.curToken.Line)
		return nil
	}

	if !p.expectPeek(token.ASSIGN) {
		return nil
	}

	p.nextToken() // move to value
	stmt.Value = p.curToken.Literal

	return stmt
}

func (p *Parser) parseComponentRef() ast.ComponentRef {
	name := p.curToken.Literal
	if p.peekTokenIs(token.DOT) {
		p.nextToken() // consume dot
		if !p.expectPeek(token.IDENT) {
			return ast.ComponentRef{Package: name}
		}
		return ast.ComponentRef{Package: name, Name: p.curToken.Literal}
	}
	return ast.ComponentRef{Name: name}
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
