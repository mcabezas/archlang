package knowledge

import (
	"errors"

	"github.com/mcabezas/archlang/graph"
)

var ErrNotFound = errors.New("not found")

const Maximum = 10000 // if we ever reach this maximum you must rethink how do you design :

type ComponentFilterOptions struct {
	NestedLevels int
	UpperLevels  int
}
type ComponentFilterOption func(*ComponentFilterOptions)

func WithNestedLevels(level int) ComponentFilterOption {
	return func(o *ComponentFilterOptions) {
		o.NestedLevels = level
	}
}

func WithUpperLevels(level int) ComponentFilterOption {
	return func(o *ComponentFilterOptions) {
		o.UpperLevels = level
	}
}

// Storage enforces to perform lazy loading by default to all their implementers
// A deep level parameter is required to receive nested components
type Storage interface {
	ListAll() ([]graph.Component, error)
	FindByName(name string, options ...ComponentFilterOption) (graph.Component, error)
	ListFeatures() ([]graph.Feature, error)
	FindByFeature(name string) ([]graph.Component, error)
	ListFlows() ([]graph.Flow, error)
	FindByFlow(name string) ([]graph.Collaboration, error)
	ListEvents() ([]graph.Component, error)
	FindEvent(name string) ([]graph.Component, error)
}
