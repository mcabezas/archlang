package generator

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mcabezas/archlang/internal/ast"
	"github.com/mcabezas/archlang/internal/lexer"
	"github.com/mcabezas/archlang/internal/parser"
)

type generateConfig struct {
	strict bool
}

type GenerateOption func(*generateConfig)

func WithStrict() GenerateOption {
	return func(c *generateConfig) { c.strict = true }
}

// Generate discovers .arch files in dir, parses them, builds the graph,
// and produces Go source code containing the hardcoded graph.
func Generate(dir string, packageName string, opts ...GenerateOption) ([]byte, error) {
	sources, err := discoverSources(dir)
	if err != nil {
		return nil, err
	}

	if len(sources) == 0 {
		return nil, fmt.Errorf("no .arch files found in %q", dir)
	}

	parsed := make(map[string]*ast.Architecture)
	for folder, input := range sources {
		l := lexer.New(input)
		p := parser.New(l)
		arch := p.Parse()

		if len(p.Errors()) > 0 {
			return nil, fmt.Errorf("parse errors in %q:\n  %s", folder, strings.Join(p.Errors(), "\n  "))
		}
		parsed[folder] = arch
	}

	cfg := &generateConfig{}
	for _, o := range opts {
		o(cfg)
	}

	g, errs := buildGraph(parsed)
	if len(errs) > 0 {
		return nil, fmt.Errorf("compile errors:\n  %s", strings.Join(errs, "\n  "))
	}

	if cfg.strict {
		for _, w := range g.warnings {
			fmt.Fprintf(os.Stderr, "%s\n", w)
		}
	}

	return generateCode(g, packageName)
}

// graphNode holds the information needed to generate code for a single node.
type graphNode struct {
	name             string
	org              string
	isService        bool
	isMessageBroker  bool
	isEvent          bool
	eventDesc        string
	messageBrokerRef string // for events: name of the message_broker they use
	brokerTechnology string // for message brokers
	cloudProvider    string // for message brokers
	platform         string // for services
	isPublic         bool
	downstreams      []string
	upstreams        []string
}

type graphEdge struct {
	sourceQN        string
	targetQN        string
	feature         string
	description     string
	cardinality     string
	cardinalityBy   string
	flow            string
	flowDescription string
	step            string
	stepOrder       int
	execute         string
	deliveredBy     string // resolved message_broker name (explicit or inherited from event)
	publishEdges    []int
}

type featureDecl struct {
	name        string
	description string
}

type brokerTechnologyDecl struct {
	name        string
	description string
}

type cloudProviderDecl struct {
	name        string
	description string
}

type platformDecl struct {
	name        string
	description string
}

// builtGraph is the intermediate representation after building from ASTs.
type builtGraph struct {
	nodes              map[string]*graphNode
	order              []string // sorted names
	edges              []graphEdge
	features           map[string]*featureDecl
	brokerTechnologies map[string]*brokerTechnologyDecl
	cloudProviders     map[string]*cloudProviderDecl
	platforms          map[string]*platformDecl
	warnings           []string
}

// builtInBrokerTechnologies lists the broker technologies pre-registered by the compiler.
var builtInBrokerTechnologies = []brokerTechnologyDecl{
	{name: "RabbitMQ", description: "RabbitMQ message broker"},
	{name: "Kafka", description: "Apache Kafka distributed event streaming platform"},
	{name: "EventBridge", description: "AWS EventBridge serverless event bus"},
}

// builtInCloudProviders lists the cloud providers pre-registered by the compiler.
var builtInCloudProviders = []cloudProviderDecl{
	{name: "AWS", description: "Amazon Web Services"},
	{name: "GCP", description: "Google Cloud Platform"},
	{name: "Azure", description: "Microsoft Azure"},
}

// builtInPlatforms lists the compute platforms pre-registered by the compiler.
var builtInPlatforms = []platformDecl{
	{name: "Lambda", description: "AWS Lambda serverless compute"},
	{name: "EKS", description: "Amazon Elastic Kubernetes Service"},
	{name: "ECS", description: "Amazon Elastic Container Service"},
	{name: "Fargate", description: "AWS Fargate serverless containers"},
	{name: "EC2", description: "Amazon Elastic Compute Cloud"},
	{name: "CloudRun", description: "Google Cloud Run serverless containers"},
	{name: "AppEngine", description: "Google App Engine"},
	{name: "AzureFunctions", description: "Azure Functions serverless compute"},
	{name: "AzureContainerApps", description: "Azure Container Apps"},
}

func buildGraph(allFolders map[string]*ast.Architecture) (*builtGraph, []string) {
	nodes := make(map[string]*graphNode)
	features := make(map[string]*featureDecl)
	brokerTechnologies := make(map[string]*brokerTechnologyDecl)
	cloudProviders := make(map[string]*cloudProviderDecl)
	platforms := make(map[string]*platformDecl)

	for _, bt := range builtInBrokerTechnologies {
		bt := bt
		brokerTechnologies[bt.name] = &bt
	}
	for _, cp := range builtInCloudProviders {
		cp := cp
		cloudProviders[cp.name] = &cp
	}
	for _, p := range builtInPlatforms {
		p := p
		platforms[p.name] = &p
	}

	var errors []string
	errors = append(errors, registerDeclarations(allFolders, nodes, features, brokerTechnologies, cloudProviders, platforms)...)
	injectDefaultBus(nodes)
	errors = append(errors, validateMessageBrokerRefs(nodes, brokerTechnologies, cloudProviders)...)
	errors = append(errors, validatePlatformRefs(nodes, platforms)...)
	edges, wireErrs := wireCollaborations(allFolders, nodes, features)
	errors = append(errors, wireErrs...)
	errors = append(errors, validateCrossOrgVisibility(nodes)...)

	order := make([]string, 0, len(nodes))
	for name := range nodes {
		order = append(order, name)
	}
	sort.Strings(order)

	// Detect orphan events (published but no subscribers)
	var warnings []string
	for _, name := range order {
		n := nodes[name]
		if n.isEvent && len(n.downstreams) == 0 {
			warnings = append(warnings, fmt.Sprintf("warning: event %q is published but has no subscribers", n.name))
		}
	}

	return &builtGraph{
		nodes:              nodes,
		order:              order,
		edges:              edges,
		features:           features,
		brokerTechnologies: brokerTechnologies,
		cloudProviders:     cloudProviders,
		platforms:          platforms,
		warnings:           warnings,
	}, errors
}

func registerDeclarations(allFolders map[string]*ast.Architecture, nodes map[string]*graphNode, features map[string]*featureDecl, brokerTechnologies map[string]*brokerTechnologyDecl, cloudProviders map[string]*cloudProviderDecl, platforms map[string]*platformDecl) []string {
	var errors []string
	for folder, arch := range allFolders {
		org := inferOrg(folder)
		for _, stmt := range arch.Statements {
			switch s := stmt.(type) {
			case *ast.ComponentStatement:
				if _, exists := nodes[s.Name]; exists {
					errors = append(errors, fmt.Sprintf("%s: line %d: duplicate declaration %q", folder, s.Token.Line, s.Name))
					continue
				}
				nodes[s.Name] = &graphNode{
					name:     s.Name,
					org:      org,
					isPublic: s.Public,
				}
			case *ast.ServiceStatement:
				if _, exists := nodes[s.Name]; exists {
					errors = append(errors, fmt.Sprintf("%s: line %d: duplicate declaration %q", folder, s.Token.Line, s.Name))
					continue
				}
				nodes[s.Name] = &graphNode{
					name:      s.Name,
					org:       org,
					isService: true,
					platform:  s.Platform,
					isPublic:  s.Public,
				}
			case *ast.BrokerTechnologyStatement:
				if _, exists := brokerTechnologies[s.Name]; exists {
					errors = append(errors, fmt.Sprintf("%s: line %d: duplicate broker_technology declaration %q", folder, s.Token.Line, s.Name))
					continue
				}
				brokerTechnologies[s.Name] = &brokerTechnologyDecl{name: s.Name, description: s.Description}
			case *ast.CloudProviderStatement:
				if _, exists := cloudProviders[s.Name]; exists {
					errors = append(errors, fmt.Sprintf("%s: line %d: duplicate cloud_provider declaration %q", folder, s.Token.Line, s.Name))
					continue
				}
				cloudProviders[s.Name] = &cloudProviderDecl{name: s.Name, description: s.Description}
			case *ast.MessageBrokerStatement:
				if _, exists := nodes[s.Name]; exists {
					errors = append(errors, fmt.Sprintf("%s: line %d: duplicate declaration %q", folder, s.Token.Line, s.Name))
					continue
				}
				nodes[s.Name] = &graphNode{
					name:             s.Name,
					org:              org,
					isMessageBroker:  true,
					brokerTechnology: s.BrokerTechnology,
					cloudProvider:    s.CloudProvider,
					isPublic:         s.Public,
				}
			case *ast.EventStatement:
				if _, exists := nodes[s.Name]; exists {
					errors = append(errors, fmt.Sprintf("%s: line %d: duplicate declaration %q", folder, s.Token.Line, s.Name))
					continue
				}
				nodes[s.Name] = &graphNode{
					name:             s.Name,
					org:              org,
					isEvent:          true,
					eventDesc:        s.Description,
					messageBrokerRef: s.MessageBroker,
				}
			case *ast.PlatformStatement:
				if _, exists := platforms[s.Name]; exists {
					errors = append(errors, fmt.Sprintf("%s: line %d: duplicate platform declaration %q", folder, s.Token.Line, s.Name))
					continue
				}
				platforms[s.Name] = &platformDecl{name: s.Name, description: s.Description}
			case *ast.FeatureStatement:
				if _, exists := features[s.Name]; exists {
					errors = append(errors, fmt.Sprintf("%s: line %d: duplicate feature declaration %q", folder, s.Token.Line, s.Name))
					continue
				}
				features[s.Name] = &featureDecl{name: s.Name, description: s.Description}
			}
		}
	}
	return errors
}

const defaultBusName = "Bus"

// injectDefaultBus assigns the built-in "Bus" broker to every event that has
// no explicit published_at. If the user declared a message_broker named "Bus"
// themselves, that declaration is used as-is and nothing is injected.
func injectDefaultBus(nodes map[string]*graphNode) {
	needsDefault := false
	for _, n := range nodes {
		if n.isEvent && n.messageBrokerRef == "" {
			needsDefault = true
			break
		}
	}
	if !needsDefault {
		return
	}

	// Ensure the Bus node exists — user may have declared it already.
	if _, exists := nodes[defaultBusName]; !exists {
		nodes[defaultBusName] = &graphNode{
			name:            defaultBusName,
			isMessageBroker: true,
		}
	}

	for _, n := range nodes {
		if n.isEvent && n.messageBrokerRef == "" {
			n.messageBrokerRef = defaultBusName
		}
	}
}

func validateMessageBrokerRefs(nodes map[string]*graphNode, brokerTechnologies map[string]*brokerTechnologyDecl, cloudProviders map[string]*cloudProviderDecl) []string {
	var errors []string
	for _, n := range nodes {
		if n.isMessageBroker {
			if n.brokerTechnology != "" {
				if _, ok := brokerTechnologies[n.brokerTechnology]; !ok {
					canonical := ""
					for k, v := range brokerTechnologies {
						if strings.EqualFold(k, n.brokerTechnology) {
							canonical = v.name
							break
						}
					}
					if canonical != "" {
						n.brokerTechnology = canonical
					} else {
						errors = append(errors, fmt.Sprintf("message_broker %q references undeclared broker_technology %q", n.name, n.brokerTechnology))
					}
				}
			}
			if n.cloudProvider != "" {
				if _, ok := cloudProviders[n.cloudProvider]; !ok {
					canonical := ""
					for k, v := range cloudProviders {
						if strings.EqualFold(k, n.cloudProvider) {
							canonical = v.name
							break
						}
					}
					if canonical != "" {
						n.cloudProvider = canonical
					} else {
						errors = append(errors, fmt.Sprintf("message_broker %q references undeclared cloud_provider %q", n.name, n.cloudProvider))
					}
				}
			}
		}
		if n.isEvent && n.messageBrokerRef != "" {
			ref, ok := nodes[n.messageBrokerRef]
			if !ok {
				errors = append(errors, fmt.Sprintf("event %q references undeclared message_broker %q", n.name, n.messageBrokerRef))
			} else if !ref.isMessageBroker {
				errors = append(errors, fmt.Sprintf("event %q: %q is not a message_broker", n.name, n.messageBrokerRef))
			}
		}
	}
	return errors
}

func validatePlatformRefs(nodes map[string]*graphNode, platforms map[string]*platformDecl) []string {
	var errors []string
	for _, n := range nodes {
		if n.isService && n.platform != "" {
			if _, ok := platforms[n.platform]; !ok {
				// Case-insensitive fallback
				canonical := ""
				for k, p := range platforms {
					if strings.EqualFold(k, n.platform) {
						canonical = p.name
						break
					}
				}
				if canonical != "" {
					n.platform = canonical
				} else {
					errors = append(errors, fmt.Sprintf("service %q references undeclared platform %q", n.name, n.platform))
				}
			}
		}
	}
	return errors
}

func wireCollaborations(allFolders map[string]*ast.Architecture, nodes map[string]*graphNode, features map[string]*featureDecl) ([]graphEdge, []string) {
	var edges []graphEdge
	var errors []string

	for folder, arch := range allFolders {
		for _, stmt := range arch.Statements {
			s, ok := stmt.(*ast.CollaborationStatement)
			if !ok {
				continue
			}

			sourceQN, err := resolveRef(s.Source, s.Token.Line, nodes)
			if err != "" {
				errors = append(errors, fmt.Sprintf("%s: %s", folder, err))
			}

			targetQN, err := resolveRef(s.Target, s.Token.Line, nodes)
			if err != "" {
				errors = append(errors, fmt.Sprintf("%s: %s", folder, err))
			}

			// Resolve delivered_by: explicit overrides, otherwise inherit from event's published_at
			deliveredBy := s.DeliveredBy
			if deliveredBy != "" {
				ref, ok := nodes[deliveredBy]
				if !ok {
					errors = append(errors, fmt.Sprintf("%s: line %d: delivered_by references undeclared message_broker %q", folder, s.Token.Line, deliveredBy))
					deliveredBy = ""
				} else if !ref.isMessageBroker {
					errors = append(errors, fmt.Sprintf("%s: line %d: delivered_by %q is not a message_broker", folder, s.Token.Line, deliveredBy))
					deliveredBy = ""
				}
			} else if sourceQN != "" {
				// inherit from the source event's published_at
				if src := nodes[sourceQN]; src != nil && src.isEvent && src.messageBrokerRef != "" {
					deliveredBy = src.messageBrokerRef
				}
			}

			if sourceQN == "" || targetQN == "" {
				continue
			}

			nodes[sourceQN].downstreams = append(nodes[sourceQN].downstreams, targetQN)
			nodes[targetQN].upstreams = append(nodes[targetQN].upstreams, sourceQN)

			if s.Feature != "" {
				if _, ok := features[s.Feature]; !ok {
					errors = append(errors, fmt.Sprintf(
						"%s: line %d: undeclared feature %q", folder, s.Token.Line, s.Feature))
				}
			}

			if s.Step != "" && s.Flow == "" {
				errors = append(errors, fmt.Sprintf(
					"%s: line %d: step %q requires a flow", folder, s.Token.Line, s.Step))
			}

			// Validate execute is only on event collaborations
			sourceIsEvent := nodes[sourceQN] != nil && nodes[sourceQN].isEvent
			targetIsEvent := nodes[targetQN] != nil && nodes[targetQN].isEvent
			if s.Execute != "" && !sourceIsEvent && !targetIsEvent {
				errors = append(errors, fmt.Sprintf(
					"%s: line %d: execute is only valid on event collaborations", folder, s.Token.Line))
			}

			subscribeIdx := len(edges)
			edges = append(edges, graphEdge{
				sourceQN:        sourceQN,
				targetQN:        targetQN,
				feature:         s.Feature,
				description:     s.Description,
				cardinality:     s.Cardinality,
				cardinalityBy:   s.CardinalityBy,
				flow:            s.Flow,
				flowDescription: s.FlowDescription,
				step:            s.Step,
				stepOrder:       s.StepOrder,
				execute:         s.Execute,
				deliveredBy:     deliveredBy,
			})

			// Expand publishes into edges and wire them to the subscribe edge
			for i, eventName := range s.Publishes {
				eventRef := ast.ComponentRef{Name: eventName}
				eventQN, err := resolveRef(eventRef, s.Token.Line, nodes)
				if err != "" {
					errors = append(errors, fmt.Sprintf("%s: %s", folder, err))
					continue
				}
				if nodes[eventQN] != nil && !nodes[eventQN].isEvent {
					errors = append(errors, fmt.Sprintf(
						"%s: line %d: publishes target %q is not an event", folder, s.Token.Line, eventName))
					continue
				}

				// The publisher is the service that subscribes and reacts.
				// After swap, for <- collaborations: source=event, target=service.
				publisherQN := targetQN

				nodes[publisherQN].downstreams = append(nodes[publisherQN].downstreams, eventQN)
				nodes[eventQN].upstreams = append(nodes[eventQN].upstreams, publisherQN)

				pubIdx := len(edges)
				edges[subscribeIdx].publishEdges = append(edges[subscribeIdx].publishEdges, pubIdx)
				edges = append(edges, graphEdge{
					sourceQN:        publisherQN,
					targetQN:        eventQN,
					feature:         s.Feature,
					description:     s.Description,
					flow:            s.Flow,
					flowDescription: s.FlowDescription,
					execute:         s.Execute,
					stepOrder:       i + 1,
				})
			}
		}
	}

	return edges, errors
}

func validateCrossOrgVisibility(nodes map[string]*graphNode) []string {
	var errors []string
	for _, n := range nodes {
		for _, dsName := range n.downstreams {
			target := nodes[dsName]
			if n.org != "" && target.org != "" && n.org != target.org && !target.isPublic {
				errors = append(errors, fmt.Sprintf(
					"%q is not public — only public components can receive calls across organizations (%s -> %s)",
					target.name, n.org, target.org))
			}
		}
	}
	return errors
}

func resolveRef(ref ast.ComponentRef, line int, nodes map[string]*graphNode) (string, string) {
	// Reconstruct the name as written
	name := ref.Name
	if ref.Domain != "" {
		name = ref.Domain + "." + ref.Name
	}

	// 1. Direct lookup by name
	if _, ok := nodes[name]; ok {
		return name, ""
	}

	// 2. Try org/name syntax: "banks/bank_api" means component "bank_api" in org "banks"
	if i := strings.LastIndex(name, "/"); i > 0 {
		orgName := name[:i]
		compName := name[i+1:]
		for nodeName, n := range nodes {
			if n.name == compName && n.org == orgName {
				return nodeName, ""
			}
		}
	}

	return "", fmt.Sprintf("line %d: undeclared %q", line, name)
}

func generateCode(g *builtGraph, packageName string) ([]byte, error) {
	var buf bytes.Buffer

	buf.WriteString("// Code generated by archlang. DO NOT EDIT.\n")
	fmt.Fprintf(&buf, "package %s\n\n", packageName)
	buf.WriteString("import \"github.com/mcabezas/archlang/graph\"\n\n")

	// Collect unique orgs
	orgSet := make(map[string]bool)
	for _, name := range g.order {
		o := g.nodes[name].org
		if o != "" {
			orgSet[o] = true
		}
	}
	var orgNames []string
	for o := range orgSet {
		orgNames = append(orgNames, o)
	}
	sort.Strings(orgNames)

	// Org constants
	if len(orgNames) > 0 {
		buf.WriteString("const (\n")
		for _, o := range orgNames {
			fmt.Fprintf(&buf, "\tOrg%s graph.Org = %q\n", toGoName(o), o)
		}
		buf.WriteString(")\n\n")
	}

	// Variable declarations
	buf.WriteString("var (\n")
	for _, name := range g.order {
		n := g.nodes[name]
		varName := toGoName(name)

		opts := fmt.Sprintf("graph.WithName(%q)", n.name)
		if n.org != "" {
			opts += fmt.Sprintf(", graph.WithOrg(Org%s)", toGoName(n.org))
		}
		if n.isPublic {
			opts += ", graph.WithVisibility(graph.Public)"
		}

		if n.isEvent {
			eventOpts := opts
			if n.messageBrokerRef != "" {
				eventOpts += fmt.Sprintf(", graph.WithMessageBrokerComponent(%s)", toGoName(n.messageBrokerRef))
			}
			fmt.Fprintf(&buf, "\t%s = graph.NewEvent(%q, %s)\n", varName, n.eventDesc, eventOpts)
		} else if n.isMessageBroker {
			fmt.Fprintf(&buf, "\t%s = graph.NewMessageBroker(%q, %q, %s)\n", varName, n.brokerTechnology, n.cloudProvider, opts)
		} else if n.isService {
			if n.platform != "" {
				opts += fmt.Sprintf(", graph.WithPlatform(%q)", n.platform)
			}
			fmt.Fprintf(&buf, "\t%s = graph.NewService(%s)\n", varName, opts)
		} else {
			fmt.Fprintf(&buf, "\t%s = graph.NewComponent(%s)\n", varName, opts)
		}
	}
	buf.WriteString(")\n\n")

	// AllComponents slice
	buf.WriteString("var AllComponents = []graph.Component{\n")
	for _, name := range g.order {
		fmt.Fprintf(&buf, "\t%s,\n", toGoName(name))
	}
	buf.WriteString("}\n\n")

	// Services slice
	var services []string
	for _, name := range g.order {
		if g.nodes[name].isService {
			services = append(services, name)
		}
	}
	buf.WriteString("var AllServices = []graph.Component{\n")
	for _, name := range services {
		fmt.Fprintf(&buf, "\t%s,\n", toGoName(name))
	}
	buf.WriteString("}\n\n")

	// Partition into connected components
	components := connectedComponents(g)

	// AllGraphs — one graph per connected component
	buf.WriteString("var AllGraphs = func() []*graph.Graph {\n")
	for i, comp := range components {
		fmt.Fprintf(&buf, "\tg%d := graph.NewGraph()\n", i)
		for _, name := range comp {
			fmt.Fprintf(&buf, "\tg%d.Register(%q, %s)\n", i, name, toGoName(name))
		}
		// Emit edges for this connected component
		compSet := make(map[string]bool)
		for _, name := range comp {
			compSet[name] = true
		}
		// Map edge index to generated variable name for wiring publishes
		edgeVars := make(map[int]string)
		edgeCounter := 0
		for ei, edge := range g.edges {
			if !compSet[edge.sourceQN] {
				continue
			}
			if edge.feature == "" && edge.execute == "" {
				fmt.Fprintf(&buf, "\tg%d.AddDownstream(%s, %s)\n", i, toGoName(edge.sourceQN), toGoName(edge.targetQN))
			} else {
				fd := g.features[edge.feature]
				featureLit := "graph.Feature{}"
				if edge.feature != "" {
					featureLit = fmt.Sprintf("graph.Feature{Name: %q", edge.feature)
					if fd != nil && fd.description != "" {
						featureLit += fmt.Sprintf(", Description: %q", fd.description)
					}
					featureLit += "}"
				}
				descLit := fmt.Sprintf("%q", edge.description)
				cardLit := fmt.Sprintf("%q", edge.cardinality)
				cardByLit := fmt.Sprintf("%q", edge.cardinalityBy)
				flowLit := "graph.Flow{}"
				if edge.flow != "" {
					flowLit = fmt.Sprintf("graph.Flow{Name: %q", edge.flow)
					if edge.flowDescription != "" {
						flowLit += fmt.Sprintf(", Description: %q", edge.flowDescription)
					}
					flowLit += "}"
				}
				stepLit := fmt.Sprintf("%q", edge.step)
				stepOrderLit := fmt.Sprintf("%d", edge.stepOrder)
				executeLit := fmt.Sprintf("%q", edge.execute)
				deliveredByLit := "nil"
				if edge.deliveredBy != "" {
					deliveredByLit = toGoName(edge.deliveredBy)
				}

				if len(edge.publishEdges) > 0 {
					varName := fmt.Sprintf("e%d", edgeCounter)
					edgeCounter++
					edgeVars[ei] = varName
					fmt.Fprintf(&buf, "\t%s := g%d.AddCollaboration(%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)\n",
						varName, i, toGoName(edge.sourceQN), toGoName(edge.targetQN), featureLit, descLit, cardLit, cardByLit, flowLit, stepLit, stepOrderLit, executeLit, deliveredByLit)
				} else {
					// Check if this edge is a publish target that needs a variable
					needsVar := false
					for _, otherEdge := range g.edges {
						for _, pubIdx := range otherEdge.publishEdges {
							if pubIdx == ei {
								needsVar = true
							}
						}
					}
					if needsVar {
						varName := fmt.Sprintf("e%d", edgeCounter)
						edgeCounter++
						edgeVars[ei] = varName
						fmt.Fprintf(&buf, "\t%s := g%d.AddCollaboration(%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)\n",
							varName, i, toGoName(edge.sourceQN), toGoName(edge.targetQN), featureLit, descLit, cardLit, cardByLit, flowLit, stepLit, stepOrderLit, executeLit, deliveredByLit)
					} else {
						fmt.Fprintf(&buf, "\tg%d.AddCollaboration(%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)\n",
							i, toGoName(edge.sourceQN), toGoName(edge.targetQN), featureLit, descLit, cardLit, cardByLit, flowLit, stepLit, stepOrderLit, executeLit, deliveredByLit)
					}
				}
			}
		}
		// Wire publish links
		for ei, edge := range g.edges {
			subVar, ok := edgeVars[ei]
			if !ok || len(edge.publishEdges) == 0 {
				continue
			}
			for _, pubIdx := range edge.publishEdges {
				if pubVar, ok := edgeVars[pubIdx]; ok {
					fmt.Fprintf(&buf, "\tg%d.LinkPublish(%s, %s)\n", i, subVar, pubVar)
				}
			}
		}
		buf.WriteString("\n")
	}
	buf.WriteString("\treturn []*graph.Graph{")
	for i := range components {
		if i > 0 {
			buf.WriteString(", ")
		}
		fmt.Fprintf(&buf, "g%d", i)
	}
	buf.WriteString("}\n")
	buf.WriteString("}()\n")

	return format.Source(buf.Bytes())
}

// connectedComponents partitions the graph into connected subgraphs.
// Edges are treated as undirected for connectivity.
func connectedComponents(g *builtGraph) [][]string {
	visited := make(map[string]bool)
	neighbors := make(map[string]map[string]bool)

	// Build undirected adjacency
	for _, name := range g.order {
		if neighbors[name] == nil {
			neighbors[name] = make(map[string]bool)
		}
		n := g.nodes[name]
		for _, dsName := range n.downstreams {
			neighbors[name][dsName] = true
			if neighbors[dsName] == nil {
				neighbors[dsName] = make(map[string]bool)
			}
			neighbors[dsName][name] = true
		}
	}

	var components [][]string
	for _, name := range g.order {
		if visited[name] {
			continue
		}
		// BFS
		var comp []string
		queue := []string{name}
		visited[name] = true
		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			comp = append(comp, cur)
			for neighbor := range neighbors[cur] {
				if !visited[neighbor] {
					visited[neighbor] = true
					queue = append(queue, neighbor)
				}
			}
		}
		sort.Strings(comp)
		components = append(components, comp)
	}

	return components
}

// inferOrg extracts the org name from a folder path.
// If the folder starts with "orgs/<name>/..." or is "orgs/<name>", the org is "<name>".
func inferOrg(folder string) string {
	if !strings.HasPrefix(folder, "orgs/") {
		return ""
	}
	rest := strings.TrimPrefix(folder, "orgs/")
	if i := strings.Index(rest, "/"); i > 0 {
		return rest[:i]
	}
	return rest
}

func toGoName(name string) string {
	name = strings.ReplaceAll(name, ".", "-")
	name = strings.ReplaceAll(name, "/", "-")
	parts := strings.Split(name, "-")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

// discoverSources walks the directory tree and returns a map of
// folder path to concatenated .arch file contents.
func discoverSources(root string) (map[string]string, error) {
	sources := make(map[string]string)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}

		input, err := readArchFiles(path)
		if err != nil {
			return err
		}
		if input == "" {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		folder := rel
		if folder == "." {
			folder = filepath.Base(root)
		}

		sources[folder] = input
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("cannot walk directory %q: %w", root, err)
	}

	return sources, nil
}

func readArchFiles(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("cannot read directory %q: %w", dir, err)
	}

	var parts []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".arch") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return "", fmt.Errorf("cannot read file %q: %w", entry.Name(), err)
		}
		parts = append(parts, string(data))
	}

	return strings.Join(parts, "\n"), nil
}
