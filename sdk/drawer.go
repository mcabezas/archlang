package knowledge

import (
	"fmt"
	"strings"

	"github.com/mcabezas/archlang/graph"
)

type mermaidDrawer struct{}

func (d *mermaidDrawer) draw(components []graph.Component) string {
	var sb strings.Builder
	sb.WriteString("graph LR\n")

	orgOrder, orgs := groupByOrg(components)
	writeSubgraphs(&sb, orgOrder, orgs)
	writeEventClassDef(&sb, components)

	type edgeKey struct{ from, to string }
	seen := make(map[edgeKey]bool)
	for _, c := range components {
		for _, collab := range c.Collaborations() {
			key := edgeKey{c.Name(), collab.Target.Name()}
			if !seen[key] {
				seen[key] = true
				arrow := "-->"
				if isEvent(collab.Source) || isEvent(collab.Target) {
					arrow = "-.->"
				}
				fmt.Fprintf(&sb, "  %s %s %s\n", nodeID(c.Name()), arrow, nodeID(collab.Target.Name()))
			}
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
			diagram.WriteString("graph LR\n")
			writeSubgraphs(&diagram, orgOrder, orgs)
			writeEventClassDef(&diagram, componentsInDiagram)

			type edgeKey struct{ from, to string }
			seen := make(map[edgeKey]bool)
			for _, collab := range collabsForDiagram {
				key := edgeKey{collab.Source.Name(), collab.Target.Name()}
				if seen[key] {
					continue
				}
				seen[key] = true
				label := edgeLabel(collab)
				arrow := "-->"
				if isEvent(collab.Source) || isEvent(collab.Target) {
					arrow = "-.->"
				}
				if label != "" {
					fmt.Fprintf(&diagram, "  %s %s|\"%s\"| %s\n", nodeID(collab.Source.Name()), arrow, label, nodeID(collab.Target.Name()))
				} else {
					fmt.Fprintf(&diagram, "  %s %s %s\n", nodeID(collab.Source.Name()), arrow, nodeID(collab.Target.Name()))
				}
			}

			fmt.Fprintf(&body, "  <pre class=\"mermaid\">\n%s  </pre>\n", diagram.String())
		}
	}

	return wrapFeatureHTML(body.String())
}

func collectComponents(collabs []graph.Collaboration) []graph.Component {
	seen := make(map[string]bool)
	var components []graph.Component
	for _, c := range collabs {
		if !seen[c.Source.Name()] {
			seen[c.Source.Name()] = true
			components = append(components, c.Source)
		}
		if !seen[c.Target.Name()] {
			seen[c.Target.Name()] = true
			components = append(components, c.Target)
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
			if isEvent(c) {
				fmt.Fprintf(sb, "    %s([\"%s\"])\n", nodeID(c.Name()), c.Name())
			} else {
				fmt.Fprintf(sb, "    %s[\"%s\"]\n", nodeID(c.Name()), c.Name())
			}
		}
		sb.WriteString("  end\n")
	}
}

func isEvent(c graph.Component) bool {
	_, ok := c.(*graph.Event)
	return ok
}

func writeEventClassDef(sb *strings.Builder, components []graph.Component) {
	var eventNodes []string
	for _, c := range components {
		if isEvent(c) {
			eventNodes = append(eventNodes, nodeID(c.Name()))
		}
	}
	if len(eventNodes) > 0 {
		sb.WriteString("  classDef event fill:#0d9488,stroke:#2dd4bf,color:#f0fdfa\n")
		fmt.Fprintf(sb, "  class %s event\n", strings.Join(eventNodes, ","))
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
      textColor: '#e2e8f0',
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
