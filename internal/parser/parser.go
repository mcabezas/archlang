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
	case token.IMPORT:
		return p.parseImportStatement()
	case token.PUBLIC, token.INTERNAL:
		return p.parseVisibilityStatement()
	case token.COMPONENT:
		return p.parseComponentStatement(false)
	case token.SERVICE:
		return p.parseServiceStatement(false)
	case token.INFRA:
		return p.parseInfraStatement(false)
	case token.COLLABORATION:
		return p.parseCollaborationStatement()
	case token.FEATURE:
		return p.parseFeatureStatement()
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

func (p *Parser) parseVisibilityStatement() ast.Statement {
	isPublic := p.curToken.Type == token.PUBLIC
	p.nextToken()
	switch p.curToken.Type {
	case token.COMPONENT:
		return p.parseComponentStatement(isPublic)
	case token.SERVICE:
		return p.parseServiceStatement(isPublic)
	case token.INFRA:
		return p.parseInfraStatement(isPublic)
	default:
		p.addError("expected component, service, or infra after visibility modifier at line %d, column %d",
			p.curToken.Line, p.curToken.Column)
		return nil
	}
}

func (p *Parser) parseComponentStatement(public bool) *ast.ComponentStatement {
	stmt := &ast.ComponentStatement{Token: p.curToken, Public: public}

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

func (p *Parser) parseServiceStatement(public bool) *ast.ServiceStatement {
	stmt := &ast.ServiceStatement{Token: p.curToken, Public: public}

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

func (p *Parser) parseInfraStatement(public bool) *ast.InfraStatement {
	stmt := &ast.InfraStatement{Token: p.curToken, Public: public}

	if !p.expectPeek(token.IDENT) {
		return nil
	}

	stmt.Name = p.curToken.Literal

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
	stmt.Domain = p.curToken.Literal
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

	// Optional block with one feature + optional description
	if p.peekTokenIs(token.LBRACE) {
		p.nextToken() // consume {
		p.parseCollaborationBlock(stmt)
	}

	return stmt
}

func (p *Parser) parseFeatureStatement() *ast.FeatureStatement {
	stmt := &ast.FeatureStatement{Token: p.curToken}

	if !p.expectPeek(token.IDENT) {
		return nil
	}
	stmt.Name = p.curToken.Literal

	if !p.expectPeek(token.COLON) {
		return nil
	}

	// Description: string literal (", ', or `)
	if !p.expectPeek(token.STRING) {
		return nil
	}
	stmt.Description = p.curToken.Literal

	return stmt
}

func (p *Parser) parseCollaborationBlock(stmt *ast.CollaborationStatement) {
	for !p.peekTokenIs(token.RBRACE) && !p.peekTokenIs(token.EOF) {
		p.nextToken()
		switch p.curToken.Type {
		case token.FEATURE:
			if stmt.Feature != "" {
				p.addError("collaboration block can only contain one feature at line %d, column %d",
					p.curToken.Line, p.curToken.Column)
				return
			}
			if !p.expectPeek(token.IDENT) {
				return
			}
			stmt.Feature = p.curToken.Literal
			// Optional inline description: feature name: "description"
			if p.peekTokenIs(token.COLON) {
				p.nextToken() // consume :
				if !p.expectPeek(token.STRING) {
					return
				}
				stmt.Description = p.curToken.Literal
			}
		case token.DESCRIPTION:
			if stmt.Description != "" {
				p.addError("collaboration block can only contain one description at line %d, column %d",
					p.curToken.Line, p.curToken.Column)
				return
			}
			if !p.expectPeek(token.COLON) {
				return
			}
			if !p.expectPeek(token.STRING) {
				return
			}
			stmt.Description = p.curToken.Literal
		case token.CARDINALITY:
			if stmt.Cardinality != "" {
				p.addError("collaboration block can only contain one cardinality at line %d, column %d",
					p.curToken.Line, p.curToken.Column)
				return
			}
			// Optional colon after cardinality keyword
			if p.peekTokenIs(token.COLON) {
				p.nextToken() // consume :
			}
			stmt.Cardinality = p.parseCardinalityValue()
			// Optional "by <name>"
			if p.peekTokenIs(token.IDENT) {
				p.nextToken()
				if p.curToken.Literal == "by" {
					if !p.expectPeek(token.IDENT) {
						return
					}
					stmt.CardinalityBy = p.curToken.Literal
				}
			}
		default:
			p.addError("expected feature, description, or cardinality, got %s at line %d, column %d",
				p.curToken.Type, p.curToken.Line, p.curToken.Column)
			return
		}
	}

	if !p.expectPeek(token.RBRACE) {
		return
	}
}

func (p *Parser) parseCardinalityValue() string {
	// "one to one" or "one to many"
	if p.peekTokenIs(token.IDENT) {
		p.nextToken()
		word := p.curToken.Literal
		if word == "one" {
			if !p.expectPeek(token.IDENT) {
				return ""
			}
			if p.curToken.Literal != "to" {
				p.addError("expected 'to' in cardinality at line %d, column %d",
					p.curToken.Line, p.curToken.Column)
				return ""
			}
			if !p.expectPeek(token.IDENT) {
				return ""
			}
			switch p.curToken.Literal {
			case "one":
				return "1:1"
			case "many":
				return "1:N"
			default:
				p.addError("expected 'one' or 'many' in cardinality at line %d, column %d",
					p.curToken.Line, p.curToken.Column)
				return ""
			}
		}
		// Single identifier like N
		p.addError("expected cardinality value (e.g. 1:1, 1:N, one to one, one to many) at line %d, column %d",
			p.curToken.Line, p.curToken.Column)
		return ""
	}
	// 1:1 or 1:N
	if !p.expectPeek(token.NUMBER) {
		return ""
	}
	left := p.curToken.Literal
	if !p.expectPeek(token.COLON) {
		return ""
	}
	if !p.peekTokenIs(token.NUMBER) && !p.peekTokenIs(token.IDENT) {
		p.addError("expected cardinality value (e.g. 1:1 or 1:N) at line %d, column %d",
			p.peekToken.Line, p.peekToken.Column)
		return ""
	}
	p.nextToken()
	return left + ":" + p.curToken.Literal
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
			return ast.ComponentRef{Domain: name}
		}
		return ast.ComponentRef{Domain: name, Name: p.curToken.Literal}
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
