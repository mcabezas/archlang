package parser

import (
	"fmt"
	"strings"

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
		if p.curTokenIs(token.FLOW) {
			arch.Statements = append(arch.Statements, p.parseFlowBlock()...)
		} else if p.curTokenIs(token.FEATURE) {
			arch.Statements = append(arch.Statements, p.parseFeatureBlock()...)
		} else {
			stmt := p.parseStatement()
			if stmt != nil {
				arch.Statements = append(arch.Statements, stmt)
			}
		}
		p.nextToken()
	}

	return arch
}

func (p *Parser) parseStatement() ast.Statement {
	switch p.curToken.Type {
	case token.PUBLIC, token.INTERNAL:
		return p.parseVisibilityStatement()
	case token.COMPONENT:
		return p.parseComponentStatement(false)
	case token.SERVICE:
		return p.parseServiceStatement(false)
	case token.MESSAGE_BROKER:
		return p.parseMessageBrokerStatement(false)
	case token.BROKER_TECHNOLOGY:
		return p.parseBrokerTechnologyStatement()
	case token.CLOUD_PROVIDER:
		return p.parseCloudProviderStatement()
	case token.PLATFORM:
		return p.parsePlatformStatement()
	case token.EVENT:
		return p.parseEventStatement()
	case token.COLLABORATION:
		return p.parseCollaborationStatement()
	case token.FEATURE:
		return nil // handled in Parse()
	case token.FLOW:
		return nil // handled in Parse()
	case token.IDENT:
		// name.attr = value (attribute assignment)
		if strings.Contains(p.curToken.Literal, ".") && p.peekTokenIs(token.ASSIGN) {
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
	case token.MESSAGE_BROKER:
		return p.parseMessageBrokerStatement(isPublic)
	default:
		p.addError("expected component, service, or message_broker after visibility modifier at line %d, column %d",
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
		p.parseComponentAttributes(&stmt.Frontend)
	}

	return stmt
}

func (p *Parser) parseServiceStatement(public bool) *ast.ServiceStatement {
	stmt := &ast.ServiceStatement{Token: p.curToken, Public: public}

	if !p.expectPeek(token.IDENT) {
		return nil
	}

	stmt.Name = p.curToken.Literal

	// Optional description
	if p.peekTokenIs(token.STRING) {
		p.nextToken()
		stmt.Description = p.curToken.Literal
	}

	if p.peekTokenIs(token.LBRACE) {
		p.nextToken() // consume {
		p.parseServiceBlock(stmt)
	}

	return stmt
}

func (p *Parser) parseServiceBlock(stmt *ast.ServiceStatement) {
	for !p.peekTokenIs(token.RBRACE) && !p.peekTokenIs(token.EOF) {
		p.nextToken()
		switch p.curToken.Type {
		case token.FRONTEND:
			stmt.Frontend = true
		case token.PLATFORM:
			if stmt.Platform != "" {
				p.addError("service block can only contain one platform at line %d", p.curToken.Line)
				return
			}
			if !p.expectPeek(token.COLON) {
				return
			}
			if !p.expectPeek(token.IDENT) {
				return
			}
			stmt.Platform = p.curToken.Literal
		default:
			p.addError("expected frontend or platform in service block, got %s at line %d",
				p.curToken.Type, p.curToken.Line)
			return
		}
	}
	if !p.expectPeek(token.RBRACE) {
		return
	}
}

func (p *Parser) parsePlatformStatement() *ast.PlatformStatement {
	stmt := &ast.PlatformStatement{Token: p.curToken}

	if !p.expectPeek(token.IDENT) {
		return nil
	}
	stmt.Name = p.curToken.Literal

	if p.peekTokenIs(token.STRING) {
		p.nextToken()
		stmt.Description = p.curToken.Literal
	}

	return stmt
}

func (p *Parser) parseBrokerTechnologyStatement() *ast.BrokerTechnologyStatement {
	stmt := &ast.BrokerTechnologyStatement{Token: p.curToken}

	if !p.expectPeek(token.IDENT) {
		return nil
	}
	stmt.Name = p.curToken.Literal

	if p.peekTokenIs(token.STRING) {
		p.nextToken()
		stmt.Description = p.curToken.Literal
	}

	return stmt
}

func (p *Parser) parseCloudProviderStatement() *ast.CloudProviderStatement {
	stmt := &ast.CloudProviderStatement{Token: p.curToken}

	if !p.expectPeek(token.IDENT) {
		return nil
	}
	stmt.Name = p.curToken.Literal

	if p.peekTokenIs(token.STRING) {
		p.nextToken()
		stmt.Description = p.curToken.Literal
	}

	return stmt
}

func (p *Parser) parseMessageBrokerStatement(public bool) *ast.MessageBrokerStatement {
	stmt := &ast.MessageBrokerStatement{Token: p.curToken, Public: public}

	if !p.expectPeek(token.IDENT) {
		return nil
	}
	stmt.Name = p.curToken.Literal

	if p.peekTokenIs(token.STRING) {
		p.nextToken()
		stmt.Description = p.curToken.Literal
	}

	if p.peekTokenIs(token.LBRACE) {
		p.nextToken() // consume {
		p.parseMessageBrokerBlock(stmt)
	}

	return stmt
}

func (p *Parser) parseMessageBrokerBlock(stmt *ast.MessageBrokerStatement) {
	for !p.peekTokenIs(token.RBRACE) && !p.peekTokenIs(token.EOF) {
		p.nextToken()
		switch p.curToken.Type {
		case token.TECHNOLOGY:
			if stmt.BrokerTechnology != "" {
				p.addError("message_broker block can only contain one technology at line %d", p.curToken.Line)
				return
			}
			if !p.expectPeek(token.COLON) {
				return
			}
			if !p.expectPeek(token.IDENT) {
				return
			}
			stmt.BrokerTechnology = p.curToken.Literal
		case token.CLOUD:
			if stmt.CloudProvider != "" {
				p.addError("message_broker block can only contain one cloud at line %d", p.curToken.Line)
				return
			}
			if !p.expectPeek(token.COLON) {
				return
			}
			if !p.expectPeek(token.IDENT) {
				return
			}
			stmt.CloudProvider = p.curToken.Literal
		default:
			p.addError("expected technology or cloud in message_broker block, got %s at line %d",
				p.curToken.Type, p.curToken.Line)
			return
		}
	}
	if !p.expectPeek(token.RBRACE) {
		return
	}
}

func (p *Parser) parseComponentAttributes(frontend *bool) {
	for !p.peekTokenIs(token.RBRACE) && !p.peekTokenIs(token.EOF) {
		p.nextToken()
		switch p.curToken.Type {
		case token.COMMA:
			continue
		case token.FRONTEND:
			*frontend = true
		default:
			p.addError("unexpected token %q in attribute block at line %d", p.curToken.Literal, p.curToken.Line)
			return
		}
	}
	if !p.expectPeek(token.RBRACE) {
		return
	}
}

func (p *Parser) parseEventStatement() *ast.EventStatement {
	stmt := &ast.EventStatement{Token: p.curToken}

	if !p.expectPeek(token.IDENT) {
		return nil
	}
	stmt.Name = p.curToken.Literal

	// Optional description
	if p.peekTokenIs(token.STRING) {
		p.nextToken()
		stmt.Description = p.curToken.Literal
	}

	// Optional block: event Name "desc" { message_broker: BrokerName }
	if p.peekTokenIs(token.LBRACE) {
		p.nextToken() // consume {
		p.parseEventBlock(stmt)
	}

	return stmt
}

func (p *Parser) parseEventBlock(stmt *ast.EventStatement) {
	for !p.peekTokenIs(token.RBRACE) && !p.peekTokenIs(token.EOF) {
		p.nextToken()
		switch p.curToken.Type {
		case token.PUBLISHED_AT:
			if stmt.MessageBroker != "" {
				p.addError("event block can only contain one published_at at line %d", p.curToken.Line)
				return
			}
			if !p.expectPeek(token.COLON) {
				return
			}
			if !p.expectPeek(token.IDENT) {
				return
			}
			stmt.MessageBroker = p.curToken.Literal
		default:
			p.addError("expected published_at in event block, got %s at line %d",
				p.curToken.Type, p.curToken.Line)
			return
		}
	}
	if !p.expectPeek(token.RBRACE) {
		return
	}
}

func (p *Parser) parseCollaborationStatement() *ast.CollaborationStatement {
	stmt := &ast.CollaborationStatement{Token: p.curToken}

	if !p.expectPeek(token.IDENT) {
		return nil
	}
	stmt.Source = p.parseComponentRef()

	// Accept -> or <-
	if p.peekTokenIs(token.ARROW) {
		p.nextToken()
	} else if p.peekTokenIs(token.REVERSE_ARROW) {
		p.nextToken()
		stmt.IsReverse = true
	} else {
		p.peekError(token.ARROW)
		return nil
	}

	if !p.expectPeek(token.IDENT) {
		return nil
	}
	stmt.Target = p.parseComponentRef()

	// For reverse arrow, swap source and target so the graph edge is target -> source
	if stmt.IsReverse {
		stmt.Source, stmt.Target = stmt.Target, stmt.Source
	}

	// Optional block with one feature + optional description
	if p.peekTokenIs(token.LBRACE) {
		p.nextToken() // consume {
		p.parseCollaborationBlock(stmt)
	}

	return stmt
}

func (p *Parser) parseFeatureBlock() []ast.Statement {
	stmt := &ast.FeatureStatement{Token: p.curToken}

	if !p.expectPeek(token.IDENT) {
		return nil
	}
	stmt.Name = p.curToken.Literal

	if !p.expectPeek(token.COLON) {
		return nil
	}

	if !p.expectPeek(token.STRING) {
		return nil
	}
	stmt.Description = p.curToken.Literal

	// If no block follows, it's a standalone feature declaration
	if !p.peekTokenIs(token.LBRACE) {
		return []ast.Statement{stmt}
	}

	p.nextToken() // consume {

	var stmts []ast.Statement
	stmts = append(stmts, stmt)

	for !p.peekTokenIs(token.RBRACE) && !p.peekTokenIs(token.EOF) {
		p.nextToken()
		switch p.curToken.Type {
		case token.COLLABORATION:
			collab := p.parseCollaborationStatement()
			if collab != nil {
				if collab.Feature != "" {
					p.addError("collaboration already belongs to feature %q, cannot be inside feature block %q at line %d, column %d",
						collab.Feature, stmt.Name, collab.Token.Line, collab.Token.Column)
				} else {
					collab.Feature = stmt.Name
				}
				stmts = append(stmts, collab)
			}
		case token.FLOW:
			flowStmts := p.parseFlowBlock()
			for _, fs := range flowStmts {
				if collab, ok := fs.(*ast.CollaborationStatement); ok {
					if collab.Feature != "" {
						p.addError("collaboration already belongs to feature %q, cannot be inside feature block %q at line %d, column %d",
							collab.Feature, stmt.Name, collab.Token.Line, collab.Token.Column)
					} else {
						collab.Feature = stmt.Name
					}
				}
				stmts = append(stmts, fs)
			}
		default:
			p.addError("expected collaboration or flow inside feature block, got %s at line %d, column %d",
				p.curToken.Type, p.curToken.Line, p.curToken.Column)
			return stmts
		}
	}

	if !p.expectPeek(token.RBRACE) {
		return stmts
	}

	return stmts
}

func (p *Parser) parseFlowBlock() []ast.Statement {
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	flowName := p.curToken.Literal

	// Optional inline description: flow name "description" { ... }
	var flowDescription string
	if p.peekTokenIs(token.STRING) {
		p.nextToken()
		flowDescription = p.curToken.Literal
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	// Optional description inside block: description: "..."
	if flowDescription == "" && p.peekTokenIs(token.DESCRIPTION) {
		p.nextToken() // consume description
		if !p.expectPeek(token.COLON) {
			return nil
		}
		if !p.expectPeek(token.STRING) {
			return nil
		}
		flowDescription = p.curToken.Literal
	}

	var stmts []ast.Statement
	stepOrderMap := make(map[string]int)
	nextOrder := 1
	for !p.peekTokenIs(token.RBRACE) && !p.peekTokenIs(token.EOF) {
		p.nextToken()
		if p.curToken.Type != token.COLLABORATION {
			p.addError("expected collaboration inside flow block, got %s at line %d, column %d",
				p.curToken.Type, p.curToken.Line, p.curToken.Column)
			return stmts
		}
		collab := p.parseCollaborationStatement()
		if collab != nil {
			if collab.Flow != "" {
				p.addError("collaboration already belongs to flow %q, cannot be inside flow %q at line %d, column %d",
					collab.Flow, flowName, collab.Token.Line, collab.Token.Column)
			} else {
				collab.Flow = flowName
				collab.FlowDescription = flowDescription
			}
			if collab.Step != "" {
				if _, seen := stepOrderMap[collab.Step]; !seen {
					stepOrderMap[collab.Step] = nextOrder
					nextOrder++
				}
				collab.StepOrder = stepOrderMap[collab.Step]
			}
			stmts = append(stmts, collab)
		}
	}

	if !p.expectPeek(token.RBRACE) {
		return stmts
	}

	return stmts
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
				if p.curToken.Literal == "by" || p.curToken.Literal == "BY" {
					if !p.expectPeek(token.IDENT) {
						return
					}
					stmt.CardinalityBy = p.curToken.Literal
				}
			}
		case token.FLOW:
			if stmt.Flow != "" {
				p.addError("collaboration already belongs to flow %q at line %d, column %d",
					stmt.Flow, p.curToken.Line, p.curToken.Column)
				return
			}
			if !p.expectPeek(token.IDENT) {
				return
			}
			stmt.Flow = p.curToken.Literal
		case token.STEP:
			if stmt.Step != "" {
				p.addError("collaboration block can only contain one step at line %d, column %d",
					p.curToken.Line, p.curToken.Column)
				return
			}
			if p.peekTokenIs(token.COLON) {
				p.nextToken() // consume :
			}
			if !p.expectPeek(token.IDENT) {
				return
			}
			stmt.Step = p.curToken.Literal
		case token.EXECUTE:
			if stmt.Execute != "" {
				p.addError("collaboration block can only contain one execute at line %d, column %d",
					p.curToken.Line, p.curToken.Column)
				return
			}
			if p.peekTokenIs(token.COLON) {
				p.nextToken() // consume :
			}
			if !p.expectPeek(token.IDENT) {
				return
			}
			stmt.Execute = p.curToken.Literal
		case token.PUBLISHES:
			if len(stmt.Publishes) > 0 {
				p.addError("collaboration block can only contain one publishes at line %d, column %d",
					p.curToken.Line, p.curToken.Column)
				return
			}
			if p.peekTokenIs(token.COLON) {
				p.nextToken() // consume :
			}
			stmt.Publishes = p.parsePublishesList()
		case token.DELIVERED_BY:
			if stmt.DeliveredBy != "" {
				p.addError("collaboration block can only contain one delivered_by at line %d, column %d",
					p.curToken.Line, p.curToken.Column)
				return
			}
			if !p.expectPeek(token.COLON) {
				return
			}
			if !p.expectPeek(token.IDENT) {
				return
			}
			stmt.DeliveredBy = p.curToken.Literal
		default:
			p.addError("expected feature, description, cardinality, flow, step, execute, publishes, or delivered_by, got %s at line %d, column %d",
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

func (p *Parser) parsePublishesList() []string {
	// Single event: publishes: EventName
	// List form: publishes: [EventA, EventB, EventC]
	if p.peekTokenIs(token.LBRACKET) {
		p.nextToken() // consume [
		var events []string
		for !p.peekTokenIs(token.RBRACKET) && !p.peekTokenIs(token.EOF) {
			if p.peekTokenIs(token.COMMA) {
				p.nextToken() // consume ,
				continue
			}
			if !p.expectPeek(token.IDENT) {
				return events
			}
			events = append(events, p.curToken.Literal)
		}
		if !p.expectPeek(token.RBRACKET) {
			return events
		}
		return events
	}

	// Single event
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	return []string{p.curToken.Literal}
}

func (p *Parser) parseAttributeStatement() *ast.AttributeStatement {
	stmt := &ast.AttributeStatement{Token: p.curToken}

	// Token literal is "component.attribute", split on first dot
	parts := strings.SplitN(p.curToken.Literal, ".", 2)
	stmt.Component = parts[0]
	stmt.Attribute = parts[1]

	if !p.expectPeek(token.ASSIGN) {
		return nil
	}

	p.nextToken() // move to value
	stmt.Value = p.curToken.Literal

	return stmt
}

func (p *Parser) parseComponentRef() ast.ComponentRef {
	name := p.curToken.Literal
	return splitComponentRef(name)
}

func splitComponentRef(name string) ast.ComponentRef {
	if i := strings.Index(name, "."); i > 0 {
		return ast.ComponentRef{Domain: name[:i], Name: name[i+1:]}
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
