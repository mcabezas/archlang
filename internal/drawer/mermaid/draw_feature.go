package mermaid

import (
	"fmt"
	"strings"

	"github.com/mcabezas/archlang/graph"
)

func (d *Drawer) DrawByFeature(components []graph.Component, feature string) string {
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
			writeClassDefs(&diagram, componentsInDiagram)

			type edgeKey struct{ from, to string }
			seen := make(map[edgeKey]bool)
			for _, collab := range collabsForDiagram {
				if collab.Target.Kind() == graph.KindEvent {
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
					if collab.DeliveredBy != nil {
						key := edgeKey{collab.DeliveredBy.Name(), collab.Target.Name()}
						if !seen[key] {
							seen[key] = true
							fmt.Fprintf(&diagram, "  %s -.->|\"listen<br>[%s]\"| %s\n", nodeID(collab.DeliveredBy.Name()), collab.Source.Name(), nodeID(collab.Target.Name()))
						}
					}
				} else {
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
