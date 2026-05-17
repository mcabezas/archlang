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
}

func (cs *ComponentStatement) statementNode()       {}
func (cs *ComponentStatement) TokenLiteral() string { return cs.Token.Literal }

type ServiceStatement struct {
	Token       token.Token
	Name        string
	Description string
	Public      bool
	Frontend    bool
	Platform    string // optional reference to a declared platform
}

type PlatformStatement struct {
	Token       token.Token
	Name        string
	Description string
}

func (ps *PlatformStatement) statementNode()       {}
func (ps *PlatformStatement) TokenLiteral() string { return ps.Token.Literal }

func (ss *ServiceStatement) statementNode()       {}
func (ss *ServiceStatement) TokenLiteral() string { return ss.Token.Literal }

type BrokerTechnologyStatement struct {
	Token       token.Token
	Name        string
	Description string
}

func (bs *BrokerTechnologyStatement) statementNode()       {}
func (bs *BrokerTechnologyStatement) TokenLiteral() string { return bs.Token.Literal }

type CloudProviderStatement struct {
	Token       token.Token
	Name        string
	Description string
}

func (cs *CloudProviderStatement) statementNode()       {}
func (cs *CloudProviderStatement) TokenLiteral() string { return cs.Token.Literal }

type MessageBrokerStatement struct {
	Token            token.Token
	Name             string
	Description      string
	BrokerTechnology string
	CloudProvider    string
	Public           bool
}

func (ms *MessageBrokerStatement) statementNode()       {}
func (ms *MessageBrokerStatement) TokenLiteral() string { return ms.Token.Literal }

type AttributeStatement struct {
	Token     token.Token
	Component string // component name
	Attribute string // "tier"
	Value     string // "0", "1", etc.
}

func (as *AttributeStatement) statementNode()       {}
func (as *AttributeStatement) TokenLiteral() string { return as.Token.Literal }

type ComponentRef struct {
	Domain string // deprecated, kept for parser compatibility
	Name    string
}

type EventStatement struct {
	Token         token.Token
	Name          string
	Description   string
	MessageBroker string // optional reference to a declared message_broker
}

func (es *EventStatement) statementNode()       {}
func (es *EventStatement) TokenLiteral() string { return es.Token.Literal }

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
	IsReverse     bool     // true if using <- operator (subscribe)
	Feature       string   // feature name (reference to declared feature), empty if none
	Description   string   // optional description of how this collaboration works
	Cardinality   string   // "1:1" or "1:N", empty if not specified
	CardinalityBy string   // partitioning key for 1:N (e.g. "account-id")
	Flow            string // flow name, empty if not inside a flow block
	FlowDescription string // flow description, set from flow block
	Step            string // step name within a flow
	StepOrder       int    // order of the step within its flow, inferred from definition order
	Execute         string   // action executed on subscribe, only valid on event collaborations
	Publishes       []string // events published as a result of this collaboration
	DeliveredBy     string   // message_broker ref for subscribe collaborations; inherits from event if omitted
}

func (cs *CollaborationStatement) statementNode()       {}
func (cs *CollaborationStatement) TokenLiteral() string { return cs.Token.Literal }
