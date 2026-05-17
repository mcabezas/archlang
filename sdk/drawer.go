package knowledge

import (
	"fmt"
	"strings"

	"github.com/mcabezas/archlang/graph"
)

type mermaidDrawer struct{}

func (d *mermaidDrawer) draw(components []graph.Component) string {
	// Collect edges and track which components participate
	type edgeKey struct{ from, to string }
	seen := make(map[edgeKey]bool)
	type edgeInfo struct {
		from, to, arrow, label string
	}
	var edges []edgeInfo
	connected := make(map[string]bool)

	for _, c := range components {
		for _, collab := range c.Collaborations() {
			if collab.Target.Kind() == graph.KindEvent {
				// Publish: service -.->|"📨 publishes EventName"| broker
				if ev, ok := collab.Target.(*graph.Event); ok && ev.MessageBroker() != nil {
					mb := ev.MessageBroker()
					key := edgeKey{c.Name(), mb.Name()}
					if !seen[key] {
						seen[key] = true
						label := "publishes<br>[<span style=\"color:#fde047;font-weight:bold\">" + collab.Target.Name() + "</span>]"
						edges = append(edges, edgeInfo{c.Name(), mb.Name(), "-.->", label})
						connected[c.Name()] = true
						connected[mb.Name()] = true
					}
				}
			} else if collab.Source.Kind() == graph.KindEvent {
				// Subscribe: broker -.->|"listen"| service
				if collab.DeliveredBy != nil {
					key := edgeKey{collab.DeliveredBy.Name(), collab.Target.Name()}
					if !seen[key] {
						seen[key] = true
						edges = append(edges, edgeInfo{collab.DeliveredBy.Name(), collab.Target.Name(), "-.->", "listen<br>[<span style=\"color:#fde047;font-weight:bold\">" + collab.Source.Name() + "</span>]"})
						connected[collab.DeliveredBy.Name()] = true
						connected[collab.Target.Name()] = true
					}
				}
			} else {
				// service --> service
				key := edgeKey{c.Name(), collab.Target.Name()}
				if !seen[key] {
					seen[key] = true
					edges = append(edges, edgeInfo{c.Name(), collab.Target.Name(), "-->", ""})
					connected[c.Name()] = true
					connected[collab.Target.Name()] = true
				}
			}
		}
	}

	// Filter to only connected components
	var active []graph.Component
	for _, c := range components {
		if connected[c.Name()] {
			active = append(active, c)
		}
	}

	var sb strings.Builder
	sb.WriteString("graph TB\n")

	orgOrder, orgs := groupByOrg(active)
	writeSubgraphs(&sb, orgOrder, orgs)
	writeEventClassDef(&sb, active)

	for _, e := range edges {
		if e.label != "" {
			fmt.Fprintf(&sb, "  %s %s|\"%s\"| %s\n", nodeID(e.from), e.arrow, e.label, nodeID(e.to))
		} else {
			fmt.Fprintf(&sb, "  %s %s %s\n", nodeID(e.from), e.arrow, nodeID(e.to))
		}
	}

	return wrapHTML("Architecture Overview", "", sb.String())
}

func (d *mermaidDrawer) drawFeature(components []graph.Component, feature string) string {
	var collabs []graph.Collaboration
	featureDesc := ""
	for _, c := range components {
		for _, collab := range c.Collaborations() {
			if collab.Feature.Name != feature {
				continue
			}
			collabs = append(collabs, collab)
			if featureDesc == "" && collab.Feature.Description != "" {
				featureDesc = collab.Feature.Description
			}
		}
	}

	type flowInfo struct {
		name        string
		description string
	}
	var flowOrder []flowInfo
	flowSeen := make(map[string]bool)
	flowSteps := make(map[string][]string)
	stepSeen := make(map[string]map[string]bool)
	stepCollabs := make(map[string]map[string][]graph.Collaboration)

	for _, c := range collabs {
		flowName := c.Flow.Name
		stepName := c.Step

		if !flowSeen[flowName] {
			flowSeen[flowName] = true
			flowOrder = append(flowOrder, flowInfo{name: flowName, description: c.Flow.Description})
			stepSeen[flowName] = make(map[string]bool)
			stepCollabs[flowName] = make(map[string][]graph.Collaboration)
		}
		if !stepSeen[flowName][stepName] {
			stepSeen[flowName][stepName] = true
			flowSteps[flowName] = append(flowSteps[flowName], stepName)
		}
		stepCollabs[flowName][stepName] = append(stepCollabs[flowName][stepName], c)
	}

	var body strings.Builder
	fmt.Fprintf(&body, "  <h1>[FEATURE] %s</h1>\n", feature)
	if featureDesc != "" {
		fmt.Fprintf(&body, "  <p>%s</p>\n", featureDesc)
	}

	for _, flow := range flowOrder {
		if flow.name != "" {
			fmt.Fprintf(&body, "  <h2>[FLOW] %s</h2>\n", flow.name)
			if flow.description != "" {
				fmt.Fprintf(&body, "  <p>%s</p>\n", flow.description)
			}
		}

		for _, step := range flowSteps[flow.name] {
			if step != "" {
				fmt.Fprintf(&body, "  <h3>[STEP] %s</h3>\n", step)
			}

			collabsForDiagram := stepCollabs[flow.name][step]
			componentsInDiagram := collectComponents(collabsForDiagram)
			orgOrder, orgs := groupByOrg(componentsInDiagram)

			var diagram strings.Builder
			diagram.WriteString("graph TB\n")
			writeSubgraphs(&diagram, orgOrder, orgs)
			writeEventClassDef(&diagram, componentsInDiagram)

			type edgeKey struct{ from, to string }
			seen := make(map[edgeKey]bool)
			for _, collab := range collabsForDiagram {
				if collab.Target.Kind() == graph.KindEvent {
					// Publish: service -.->|"📨 publishes EventName"| broker
					if ev, ok := collab.Target.(*graph.Event); ok && ev.MessageBroker() != nil {
						mb := ev.MessageBroker()
						key := edgeKey{collab.Source.Name(), mb.Name()}
						if !seen[key] {
							seen[key] = true
							label := "publishes<br>[<span style=\"color:#fde047;font-weight:bold\">" + collab.Target.Name() + "</span>]"
							fmt.Fprintf(&diagram, "  %s -.->|\"%s\"| %s\n", nodeID(collab.Source.Name()), label, nodeID(mb.Name()))
						}
					}
				} else if collab.Source.Kind() == graph.KindEvent {
					// Subscribe: broker -.->|"listen"| service
					if collab.DeliveredBy != nil {
						key := edgeKey{collab.DeliveredBy.Name(), collab.Target.Name()}
						if !seen[key] {
							seen[key] = true
							fmt.Fprintf(&diagram, "  %s -.->|\"listen<br>[%s]\"| %s\n", nodeID(collab.DeliveredBy.Name()), collab.Source.Name(), nodeID(collab.Target.Name()))
						}
					}
				} else {
					// service --> service
					key := edgeKey{collab.Source.Name(), collab.Target.Name()}
					if !seen[key] {
						seen[key] = true
						label := edgeLabel(collab)
						if label != "" {
							fmt.Fprintf(&diagram, "  %s -->|\"%s\"| %s\n", nodeID(collab.Source.Name()), label, nodeID(collab.Target.Name()))
						} else {
							fmt.Fprintf(&diagram, "  %s --> %s\n", nodeID(collab.Source.Name()), nodeID(collab.Target.Name()))
						}
					}
				}
			}

			fmt.Fprintf(&body, "  <pre class=\"mermaid\">\n%s  </pre>\n", diagram.String())
		}
	}

	return wrapFeatureHTML(body.String())
}

func (d *mermaidDrawer) drawEvent(components []graph.Component, eventName string) string {
	type edgeKey struct{ from, to string }
	seen := make(map[edgeKey]bool)
	type edgeInfo struct{ from, to, arrow, label string }
	var edges []edgeInfo
	connected := make(map[string]bool)

	for _, c := range components {
		for _, collab := range c.Collaborations() {
			if collab.Target.Kind() == graph.KindEvent && collab.Target.Name() == eventName {
				if ev, ok := collab.Target.(*graph.Event); ok && ev.MessageBroker() != nil {
					mb := ev.MessageBroker()
					key := edgeKey{c.Name(), mb.Name()}
					if !seen[key] {
						seen[key] = true
						label := "publishes<br>[<span style=\"color:#fde047;font-weight:bold\">" + collab.Target.Name() + "</span>]"
						edges = append(edges, edgeInfo{c.Name(), mb.Name(), "-.->", label})
						connected[c.Name()] = true
						connected[mb.Name()] = true
					}
				}
			}
			if collab.Source.Kind() == graph.KindEvent && collab.Source.Name() == eventName && collab.DeliveredBy != nil {
				key := edgeKey{collab.DeliveredBy.Name(), collab.Target.Name()}
				if !seen[key] {
					seen[key] = true
					label := "listen<br>[<span style=\"color:#fde047;font-weight:bold\">" + collab.Source.Name() + "</span>]"
					edges = append(edges, edgeInfo{collab.DeliveredBy.Name(), collab.Target.Name(), "-.->", label})
					connected[collab.DeliveredBy.Name()] = true
					connected[collab.Target.Name()] = true
				}
			}
		}
	}

	var active []graph.Component
	for _, c := range components {
		if connected[c.Name()] {
			active = append(active, c)
		}
	}

	var sb strings.Builder
	sb.WriteString("graph TB\n")
	orgOrder, orgs := groupByOrg(active)
	writeSubgraphs(&sb, orgOrder, orgs)
	writeEventClassDef(&sb, active)

	for _, e := range edges {
		fmt.Fprintf(&sb, "  %s %s|\"%s\"| %s\n", nodeID(e.from), e.arrow, e.label, nodeID(e.to))
	}

	return wrapHTML("Event: "+eventName, "", sb.String())
}

func collectComponents(collabs []graph.Collaboration) []graph.Component {
	seen := make(map[string]bool)
	var components []graph.Component
	add := func(c graph.Component) {
		if !seen[c.Name()] {
			seen[c.Name()] = true
			components = append(components, c)
		}
	}
	for _, c := range collabs {
		if c.Target.Kind() == graph.KindEvent {
			// Publish: include source service and broker, skip event node
			add(c.Source)
			if ev, ok := c.Target.(*graph.Event); ok && ev.MessageBroker() != nil {
				add(ev.MessageBroker())
			}
		} else if c.Source.Kind() == graph.KindEvent {
			// Subscribe: include broker and target service, skip event node
			if c.DeliveredBy != nil {
				add(c.DeliveredBy)
			}
			add(c.Target)
		} else {
			add(c.Source)
			add(c.Target)
		}
	}
	return components
}

func groupByOrg(components []graph.Component) ([]graph.Org, map[graph.Org][]graph.Component) {
	orgs := make(map[graph.Org][]graph.Component)
	var orgOrder []graph.Org
	for _, c := range components {
		org := c.Org()
		if _, exists := orgs[org]; !exists {
			orgOrder = append(orgOrder, org)
		}
		orgs[org] = append(orgs[org], c)
	}
	return orgOrder, orgs
}

func writeSubgraphs(sb *strings.Builder, orgOrder []graph.Org, orgs map[graph.Org][]graph.Component) {
	for _, org := range orgOrder {
		fmt.Fprintf(sb, "  subgraph org_%s [\"%s\"]\n", nodeID(string(org)), string(org))
		for _, c := range orgs[org] {
			switch {
			case c.Kind() == graph.KindEvent:
				fmt.Fprintf(sb, "    %s([\"Event: %s\"])\n", nodeID(c.Name()), c.Name())
			case c.Kind() == graph.KindMessageBroker:
				fmt.Fprintf(sb, "    %s[(\"✉️✉️✉️✉️✉️✉️<br>✉️ %s ✉️<br>✉️✉️✉️✉️✉️✉️\")]\n", nodeID(c.Name()), c.Name())
			default:
				fmt.Fprintf(sb, "    %s[\"%s\"]\n", nodeID(c.Name()), c.Name())
			}
		}
		sb.WriteString("  end\n")
	}
}


func writeEventClassDef(sb *strings.Builder, components []graph.Component) {
	var eventNodes []string
	var brokerNodes []string
	var serviceNodes []string
	for _, c := range components {
		switch c.Kind() {
		case graph.KindEvent:
			eventNodes = append(eventNodes, nodeID(c.Name()))
		case graph.KindMessageBroker:
			brokerNodes = append(brokerNodes, nodeID(c.Name()))
		default:
			serviceNodes = append(serviceNodes, nodeID(c.Name()))
		}
	}
	if len(serviceNodes) > 0 {
		sb.WriteString("  classDef service fill:#1e293b,stroke:#38bdf8,color:#fde047,font-weight:bold\n")
		fmt.Fprintf(sb, "  class %s service\n", strings.Join(serviceNodes, ","))
	}
	if len(eventNodes) > 0 {
		sb.WriteString("  classDef event fill:#0d9488,stroke:#2dd4bf,color:#fde047,font-weight:bold\n")
		fmt.Fprintf(sb, "  class %s event\n", strings.Join(eventNodes, ","))
	}
	if len(brokerNodes) > 0 {
		sb.WriteString("  classDef messageBroker fill:#7c3aed,stroke:#a78bfa,color:#f5f3ff,font-weight:bold\n")
		fmt.Fprintf(sb, "  class %s messageBroker\n", strings.Join(brokerNodes, ","))
		for _, n := range brokerNodes {
			fmt.Fprintf(sb, "  style %s min-width:300px\n", n)
		}
	}
}

func edgeLabel(c graph.Collaboration) string {
	var parts []string
	if c.Execute != "" {
		parts = append(parts, c.Execute+"()")
	}
	if c.Description != "" {
		parts = append(parts, c.Description)
	}
	if c.Cardinality != "" && c.Cardinality != "1:1" {
		card := c.Cardinality
		if c.CardinalityBy != "" {
			card += " by " + c.CardinalityBy
		}
		parts = append(parts, card)
	}
	return strings.Join(parts, "<br>")
}

func nodeID(name string) string {
	return strings.ReplaceAll(name, "-", "_")
}

func toUpperSnake(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, []rune(strings.ToUpper(string(r)))...)
	}
	return string(result)
}

const mermaidInit = `<script src="https://cdn.jsdelivr.net/npm/mermaid/dist/mermaid.min.js"></script>
  <script>mermaid.initialize({
    startOnLoad: true,
    theme: 'base',
    themeVariables: {
      background: '#1e293b',
      primaryColor: '#1e293b',
      primaryTextColor: '#e2e8f0',
      primaryBorderColor: '#38bdf8',
      lineColor: '#64748b',
      secondaryColor: '#334155',
      tertiaryColor: '#0f172a',
      textColor: '#fde047',
      mainBkg: '#1e293b',
      nodeBorder: '#38bdf8',
      clusterBkg: '#0f172a',
      clusterBorder: '#334155',
      edgeLabelBackground: '#1e293b',
      fontFamily: '-apple-system, BlinkMacSystemFont, Segoe UI, Roboto, sans-serif',
      fontSize: '14px'
    }
  });</script>`

const darkCSS = `<style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #0f172a; color: #e2e8f0; padding: 2rem 3rem; line-height: 1.6; }
    h1 { font-size: 2rem; font-weight: 700; color: #f8fafc; margin-bottom: 0.25rem; padding-bottom: 1rem; border-bottom: 1px solid #1e293b; }
    h1 + p { color: #94a3b8; font-size: 1.05rem; margin-bottom: 2rem; }
    h2 { font-size: 1.35rem; font-weight: 600; color: #38bdf8; margin-top: 2.5rem; margin-bottom: 0.25rem; text-transform: uppercase; letter-spacing: 0.05em; }
    h2 + p { color: #94a3b8; font-size: 0.95rem; margin-bottom: 1rem; }
    h3 { font-size: 1.05rem; font-weight: 500; color: #a78bfa; margin-top: 1.5rem; margin-bottom: 0.75rem; padding-left: 0.75rem; border-left: 3px solid #a78bfa; }
    pre.mermaid { background: #1e293b; border-radius: 12px; padding: 1.5rem; margin: 0.5rem 0 1.5rem; border: 1px solid #334155; }
  </style>`

func wrapHTML(title string, subtitle string, mermaidCode string) string {
	subtitleHTML := ""
	if subtitle != "" {
		subtitleHTML = fmt.Sprintf("\n  <p>%s</p>", subtitle)
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>%s</title>
  %s
  %s
</head>
<body>
  <h1>%s</h1>%s
  <pre class="mermaid">
%s
  </pre>
</body>
</html>`, title, mermaidInit, darkCSS, title, subtitleHTML, mermaidCode)
}

func wrapFeatureHTML(body string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>Feature</title>
  %s
  %s
</head>
<body>
%s
</body>
</html>`, mermaidInit, darkCSS, body)
}
