package mermaid

import (
	"fmt"
	"strings"

	"github.com/mcabezas/archlang/graph"
)

func (d *Drawer) DrawByEvent(components []graph.Component, eventName string) string {
	var participantOrder []string
	participants := make(map[string]string) // name -> display label

	addParticipant := func(c graph.Component) {
		name := c.Name()
		if _, exists := participants[name]; !exists {
			label := name
			if svc, ok := c.(*graph.Service); ok {
				switch {
				case strings.EqualFold(svc.Platform, "lambda"):
					label = name + " ⚡λ"
				case strings.EqualFold(svc.Platform, "ecs"):
					label = name + " 🐳"
				case strings.EqualFold(svc.Platform, "eks"):
					label = name + " ☸️"
				}
			} else if _, ok := c.(*graph.MessageBroker); ok {
				label = "✉️ " + name + " 🔀"
			}
			participants[name] = label
			participantOrder = append(participantOrder, name)
		}
	}

	// Pass 1: collect participants in causal order (publishers first, then broker, then subscribers)
	for _, c := range components {
		for _, collab := range c.Collaborations() {
			if collab.Target.Kind() == graph.KindEvent && collab.Target.Name() == eventName {
				if ev, ok := collab.Target.(*graph.Event); ok && ev.MessageBroker() != nil {
					addParticipant(c)
					addParticipant(ev.MessageBroker())
				}
			}
		}
	}
	for _, c := range components {
		for _, collab := range c.Collaborations() {
			if collab.Source.Kind() == graph.KindEvent && collab.Source.Name() == eventName && collab.DeliveredBy != nil {
				addParticipant(collab.DeliveredBy)
				addParticipant(collab.Target)
				for _, pub := range collab.Publishes {
					if pub.Target.Kind() == graph.KindEvent {
						if ev, ok := pub.Target.(*graph.Event); ok && ev.MessageBroker() != nil {
							addParticipant(pub.Source)
							addParticipant(ev.MessageBroker())
						}
					}
				}
			}
		}
	}

	var lines []string
	seenEdge := make(map[string]bool)

	addLine := func(s string) { lines = append(lines, s) }

	// Pass 2: build diagram lines in causal order
	// Publishers
	for _, c := range components {
		for _, collab := range c.Collaborations() {
			if collab.Target.Kind() == graph.KindEvent && collab.Target.Name() == eventName {
				if ev, ok := collab.Target.(*graph.Event); ok && ev.MessageBroker() != nil {
					mb := ev.MessageBroker()
					key := c.Name() + "->" + mb.Name() + ":publishes:" + eventName
					if seenEdge[key] {
						continue
					}
					seenEdge[key] = true
					execute := collab.Execute
					if execute == "" {
						execute = "execute"
					}
					addLine(fmt.Sprintf("  activate %s", nodeID(c.Name())))
					addLine(fmt.Sprintf("  Note over %s: %s()", nodeID(c.Name()), execute))
					addLine(fmt.Sprintf("  %s->>%s: publishes [%s]", nodeID(c.Name()), nodeID(mb.Name()), eventName))
					addLine(fmt.Sprintf("  deactivate %s", nodeID(c.Name())))
				}
			}
		}
	}

	// Subscribers + caused-by
	for _, c := range components {
		for _, collab := range c.Collaborations() {
			if collab.Source.Kind() == graph.KindEvent && collab.Source.Name() == eventName && collab.DeliveredBy != nil {
				key := collab.DeliveredBy.Name() + "->" + collab.Target.Name() + ":listen:" + eventName
				if seenEdge[key] {
					continue
				}
				seenEdge[key] = true

				execute := collab.Execute
				if execute == "" {
					execute = "execute"
				}
				addLine(fmt.Sprintf("  %s-->>%s: listen [%s]", nodeID(collab.DeliveredBy.Name()), nodeID(collab.Target.Name()), eventName))
				addLine(fmt.Sprintf("  activate %s", nodeID(collab.Target.Name())))
				addLine(fmt.Sprintf("  Note right of %s: %s()", nodeID(collab.Target.Name()), execute))
				for _, pub := range collab.Publishes {
					if pub.Target.Kind() == graph.KindEvent {
						if ev, ok := pub.Target.(*graph.Event); ok && ev.MessageBroker() != nil {
							mb := ev.MessageBroker()
							pubLabel := "publishes [" + pub.Target.Name() + "]"
							if pub.Execute != "" {
								pubLabel = pub.Execute + "()<br>" + pubLabel
							}
							addLine(fmt.Sprintf("  %s->>%s: %s", nodeID(pub.Source.Name()), nodeID(mb.Name()), pubLabel))
						}
					}
				}
				addLine(fmt.Sprintf("  deactivate %s", nodeID(collab.Target.Name())))
			}
		}
	}

	var sb strings.Builder
	sb.WriteString("sequenceDiagram\n")
	for _, name := range participantOrder {
		fmt.Fprintf(&sb, "  participant %s as %s\n", nodeID(name), participants[name])
	}
	sb.WriteString("\n")
	for _, line := range lines {
		sb.WriteString(line + "\n")
	}

	return wrapHTML("Event: "+eventName, "", sb.String())
}
