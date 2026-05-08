package ast

import (
	"github.com/mcabezas/archlang/internal/token"
)

type Node interface {
	TokenLiteral() string
}

type Statement interface {
	Node
	statementNode()
}

type Architecture struct {
	Statements []Statement
}

func (a *Architecture) TokenLiteral() string {
	if len(a.Statements) > 0 {
		return a.Statements[0].TokenLiteral()
	}
	return ""
}

type ComponentStatement struct {
	Token token.Token
	Name  string
}

func (cs *ComponentStatement) statementNode()       {}
func (cs *ComponentStatement) TokenLiteral() string { return cs.Token.Literal }

type ServiceStatement struct {
	Token token.Token
	Name  string
}

func (ss *ServiceStatement) statementNode()       {}
func (ss *ServiceStatement) TokenLiteral() string { return ss.Token.Literal }

type CollaborationStatement struct {
	Token  token.Token
	Source string
	Target string
}

func (cs *CollaborationStatement) statementNode()       {}
func (cs *CollaborationStatement) TokenLiteral() string { return cs.Token.Literal }
