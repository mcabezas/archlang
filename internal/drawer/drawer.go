package drawer

import "github.com/mcabezas/archlang/graph"

// Drawer renders architecture components as an HTML page with embedded diagrams.
type Drawer interface {
	Draw(components []graph.Component) string
	DrawByFeature(components []graph.Component, feature string) string
	DrawByEvent(components []graph.Component, eventName string) string
}
