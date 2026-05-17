package graph

const DomainDefault Domain = "default"

// ComponentKind identifies the role of a component in the architecture.
type ComponentKind string

const (
	KindService       ComponentKind = "service"
	KindMessageBroker ComponentKind = "message_broker"
	KindEvent         ComponentKind = "event"
	KindComponent     ComponentKind = "component"
)

// IsInfra reports whether this kind represents infrastructure.
func (k ComponentKind) IsInfra() bool {
	return k == KindMessageBroker
}

// Component is the base type for all elements in the architecture graph.
type Component interface {
	Name() string
	Kind() ComponentKind
	Collaborations() []Collaboration
	Domain() Domain
	Org() Org
	Visibility() Visibility
	Base() Component
}

type Service struct {
	Component
	RepositoryURL string
	Platform      string
}

func (s *Service) Kind() ComponentKind { return KindService }

type Infra struct{ Component }

type MessageBroker struct {
	Component
	BrokerTechnology string
	CloudProvider    string
}

func (m *MessageBroker) Kind() ComponentKind { return KindMessageBroker }

type Event struct {
	Component
	description   string
	messageBroker *MessageBroker
}

func (e *Event) Kind() ComponentKind          { return KindEvent }
func (e *Event) Description() string          { return e.description }
func (e *Event) MessageBroker() *MessageBroker { return e.messageBroker }

type Domain string
type Org string

type Feature struct {
	Name        string
	Description string
}

type Flow struct {
	Name        string
	Description string
}

type Collaboration struct {
	Source        Component
	Target        Component
	Feature       Feature
	Description   string
	Cardinality   string
	CardinalityBy string
	Flow          Flow
	Step          string
	StepOrder     int
	Execute       string
	Publishes     []*Collaboration
	DeliveredBy   *MessageBroker // resolved at compile time; inherited from event's published_at if not explicit
}

type Visibility string

const (
	Internal Visibility = "internal"
	Public   Visibility = "public"
)

type component struct {
	name           string
	domain         Domain
	org            Org
	visibility     Visibility
	collaborations []Collaboration
}

type NewComponentOptions struct {
	Name          string
	Domain        Domain
	Org           Org
	Visibility    Visibility
	RepositoryURL string
	MessageBroker *MessageBroker
	Platform      string
}

type NewComponentOption func(*NewComponentOptions)

func WithName(name string) NewComponentOption {
	return func(o *NewComponentOptions) {
		o.Name = name
	}
}

func WithDomain(domain Domain) NewComponentOption {
	return func(o *NewComponentOptions) {
		o.Domain = domain
	}
}

func WithOrg(org Org) NewComponentOption {
	return func(o *NewComponentOptions) {
		o.Org = org
	}
}

func WithVisibility(v Visibility) NewComponentOption {
	return func(o *NewComponentOptions) {
		o.Visibility = v
	}
}

func WithRepositoryURL(url string) NewComponentOption {
	return func(o *NewComponentOptions) {
		o.RepositoryURL = url
	}
}

func WithMessageBrokerComponent(mb *MessageBroker) NewComponentOption {
	return func(o *NewComponentOptions) {
		o.MessageBroker = mb
	}
}

func WithPlatform(platform string) NewComponentOption {
	return func(o *NewComponentOptions) {
		o.Platform = platform
	}
}

func NewComponent(options ...NewComponentOption) Component {
	opts := &NewComponentOptions{}
	for _, option := range options {
		option(opts)
	}

	return &component{name: opts.Name, domain: opts.Domain, org: opts.Org, visibility: opts.visibility()}
}

func NewService(options ...NewComponentOption) *Service {
	opts := &NewComponentOptions{}
	for _, option := range options {
		option(opts)
	}

	return &Service{
		Component:     &component{name: opts.Name, domain: opts.Domain, org: opts.Org, visibility: opts.visibility()},
		RepositoryURL: opts.RepositoryURL,
		Platform:      opts.Platform,
	}
}

func NewEvent(description string, options ...NewComponentOption) *Event {
	opts := &NewComponentOptions{}
	for _, option := range options {
		option(opts)
	}

	return &Event{
		Component:     &component{name: opts.Name, domain: opts.Domain, org: opts.Org, visibility: opts.visibility()},
		description:   description,
		messageBroker: opts.MessageBroker,
	}
}

func NewMessageBroker(brokerTechnology, cloudProvider string, options ...NewComponentOption) *MessageBroker {
	opts := &NewComponentOptions{}
	for _, option := range options {
		option(opts)
	}

	return &MessageBroker{
		Component:        &component{name: opts.Name, domain: opts.Domain, org: opts.Org, visibility: opts.visibility()},
		BrokerTechnology: brokerTechnology,
		CloudProvider:    cloudProvider,
	}
}

func NewInfra(options ...NewComponentOption) *Infra {
	opts := &NewComponentOptions{}
	for _, option := range options {
		option(opts)
	}

	return &Infra{
		Component: &component{name: opts.Name, domain: opts.Domain, org: opts.Org, visibility: opts.visibility()},
	}
}

func (o *NewComponentOptions) visibility() Visibility {
	if o.Visibility == "" {
		return Internal
	}
	return o.Visibility
}

func (n *component) Name() string {
	return n.name
}

func (n *component) Kind() ComponentKind {
	return KindComponent
}

func (n *component) Base() Component {
	return n
}

func (n *component) Collaborations() []Collaboration {
	return n.collaborations
}

func (n *component) Domain() Domain {
	return n.domain
}

func (n *component) Org() Org {
	return n.org
}

func (n *component) Visibility() Visibility {
	return n.visibility
}
