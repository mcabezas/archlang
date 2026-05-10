package graph

const DomainDefault Domain = "default"

// Component is the base type for all elements in the architecture graph.
type Component interface {
	Name() string
	Downstreams() []Component
	Upstreams() []Component
	Domain() Domain
	Visibility() Visibility
	Base() Component
}

type Service struct {
	Component
	RepositoryURL string
}
type Infra struct{ Component }

type Domain string
type Org string

type Visibility string

const (
	Internal Visibility = "internal"
	Public   Visibility = "public"
)

type component struct {
	name        string
	domain      Domain
	org         Org
	visibility  Visibility
	downstreams []Component
	upstreams   []Component
}

type NewComponentOptions struct {
	Name          string
	Domain        Domain
	Org           Org
	Visibility    Visibility
	RepositoryURL string
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

func (n *component) Base() Component {
	return n
}

func (n *component) Downstreams() []Component {
	return n.downstreams
}

func (n *component) Upstreams() []Component {
	return n.upstreams
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
