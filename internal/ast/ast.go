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
	Token    token.Token
	Name     string
	Public   bool
	Frontend bool
	Infra    string // "db", "cache", "bus", "lb", or "" for generic
}

func (cs *ComponentStatement) statementNode()       {}
func (cs *ComponentStatement) TokenLiteral() string { return cs.Token.Literal }

type ServiceStatement struct {
	Token    token.Token
	Name     string
	Public   bool
	Frontend bool
}

func (ss *ServiceStatement) statementNode()       {}
func (ss *ServiceStatement) TokenLiteral() string { return ss.Token.Literal }

type InfraStatement struct {
	Token  token.Token
	Name   string
	Public bool
}

func (is *InfraStatement) statementNode()       {}
func (is *InfraStatement) TokenLiteral() string { return is.Token.Literal }

type AttributeStatement struct {
	Token     token.Token
	Component string // component name
	Attribute string // "tier"
	Value     string // "0", "1", etc.
}

func (as *AttributeStatement) statementNode()       {}
func (as *AttributeStatement) TokenLiteral() string { return as.Token.Literal }

type ImportStatement struct {
	Token   token.Token
	Domain string
	Alias  string
}

func (is *ImportStatement) statementNode()       {}
func (is *ImportStatement) TokenLiteral() string { return is.Token.Literal }

type ComponentRef struct {
	Domain string // empty for local references
	Name    string
}

type FeatureStatement struct {
	Token       token.Token
	Name        string
	Description string
}

func (fs *FeatureStatement) statementNode()       {}
func (fs *FeatureStatement) TokenLiteral() string { return fs.Token.Literal }

type CollaborationStatement struct {
	Token         token.Token
	Source        ComponentRef
	Target        ComponentRef
	Feature       string // feature name (reference to declared feature), empty if none
	Description   string // optional description of how this collaboration works
	Cardinality   string // "1:1" or "1:N", empty if not specified
	CardinalityBy string // partitioning key for 1:N (e.g. "account-id")
	Flow            string // flow name, empty if not inside a flow block
	FlowDescription string // flow description, set from flow block
	Step            string // step name within a flow
	StepOrder       int    // order of the step within its flow, inferred from definition order
}

func (cs *CollaborationStatement) statementNode()       {}
func (cs *CollaborationStatement) TokenLiteral() string { return cs.Token.Literal }
