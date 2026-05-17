package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mcabezas/archlang/graph"
	"github.com/mcabezas/archlang/internal/drawer"
	"github.com/mcabezas/archlang/internal/drawer/mermaid"
)

// HTTPServer serves the architecture graph over HTTP.
type HTTPServer struct {
	storage Storage
	drawer  drawer.Drawer
	addr    string
}

// NewHTTPServer creates a new HTTP server that serves the architecture graph.
func NewHTTPServer(graphs []*graph.Graph, addr string) *HTTPServer {
	return &HTTPServer{
		storage: New(graphs),
		drawer:  mermaid.New(),
		addr:    addr,
	}
}

// Start begins listening and serving HTTP requests.
// It blocks until an interrupt signal (SIGINT, SIGTERM) is received,
// then gracefully shuts down allowing in-flight requests to complete.
func (s *HTTPServer) Start() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/components/", s.handleGetComponents)
	mux.HandleFunc("/api/events/", s.handleGetEvent)
	mux.HandleFunc("/api/events", s.handleListEvents)
	mux.HandleFunc("/api/features", s.handleListFeatures)
	mux.HandleFunc("/api/graph", s.handleGraph)
	mux.HandleFunc("/diagram", s.handleDiagram)
	mux.HandleFunc("/alpha", s.handleAlpha)

	srv := &http.Server{
		Addr:    s.addr,
		Handler: mux,
	}

	errCh := make(chan error, 1)
	go func() {
		fmt.Printf("http server listening on %s\n", s.addr)
		errCh <- srv.ListenAndServe()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		fmt.Printf("\nreceived %s, shutting down...\n", sig)
	case err := <-errCh:
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown error: %w", err)
	}

	fmt.Println("server stopped")
	return nil
}

func (s *HTTPServer) handleGetComponents(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/components/")
	if name == "" {
		http.Error(w, "component name required", http.StatusBadRequest)
		return
	}

	c, err := s.storage.FindByName(name)
	if err != nil {
		http.Error(w, "component not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toComponentJSON(c))
}

func (s *HTTPServer) handleListEvents(w http.ResponseWriter, r *http.Request) {
	events, err := s.storage.ListEvents()
	if err != nil {
		http.Error(w, "failed to list events", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	var result []componentJSON
	for _, e := range events {
		result = append(result, toComponentJSON(e))
	}
	json.NewEncoder(w).Encode(result)
}

func (s *HTTPServer) handleListFeatures(w http.ResponseWriter, r *http.Request) {
	features, err := s.storage.ListFeatures()
	if err != nil {
		http.Error(w, "failed to list features", http.StatusInternalServerError)
		return
	}
	type featureJSON struct {
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
	}
	var result []featureJSON
	for _, f := range features {
		result = append(result, featureJSON{Name: f.Name, Description: f.Description})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *HTTPServer) handleGetEvent(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/events/")
	if name == "" {
		http.Error(w, "event name required", http.StatusBadRequest)
		return
	}
	components, err := s.storage.FindEvent(name)
	if err != nil {
		http.Error(w, "event not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	var result []componentJSON
	for _, c := range components {
		result = append(result, toComponentJSON(c))
	}
	json.NewEncoder(w).Encode(result)
}

func (s *HTTPServer) handleDiagram(w http.ResponseWriter, r *http.Request) {
	feature := r.URL.Query().Get("feature")
	event := r.URL.Query().Get("event")

	w.Header().Set("Content-Type", "text/html")

	if feature != "" {
		components, err := s.storage.FindByFeature(feature)
		if err != nil {
			http.Error(w, "feature not found", http.StatusNotFound)
			return
		}
		fmt.Fprint(w, s.drawer.DrawByFeature(components, feature))
		return
	}

	if event != "" {
		components, err := s.storage.FindEvent(event)
		if err != nil {
			http.Error(w, "event not found", http.StatusNotFound)
			return
		}
		fmt.Fprint(w, s.drawer.DrawByEvent(components, event))
		return
	}

	all, err := s.storage.ListAll()
	if err != nil {
		http.Error(w, "failed to list components", http.StatusInternalServerError)
		return
	}
	fmt.Fprint(w, s.drawer.Draw(all))
}

type brokerJSON struct {
	Name       string `json:"name"`
	Technology string `json:"technology,omitempty"`
	Cloud      string `json:"cloud,omitempty"`
}

type componentJSON struct {
	Name             string              `json:"name"`
	Domain           string              `json:"domain,omitempty"`
	Kind             string              `json:"kind"`
	Visibility       string              `json:"visibility"`
	Platform         string              `json:"platform,omitempty"`
	BrokerTechnology string              `json:"broker_technology,omitempty"`
	CloudProvider    string              `json:"cloud_provider,omitempty"`
	PublishedAt      *brokerJSON         `json:"published_at,omitempty"`
	Collaborations   []collaborationJSON `json:"collaborations"`
}

type collaborationJSON struct {
	Target        string `json:"target"`
	Feature       string `json:"feature,omitempty"`
	Description   string `json:"description,omitempty"`
	Cardinality   string `json:"cardinality,omitempty"`
	CardinalityBy string `json:"cardinality_by,omitempty"`
	Flow          string `json:"flow,omitempty"`
	Step          string `json:"step,omitempty"`
	StepOrder     int    `json:"step_order,omitempty"`
	DeliveredBy   string `json:"delivered_by,omitempty"`
}

func toComponentJSON(c graph.Component) componentJSON {
	var collabs []collaborationJSON
	for _, col := range c.Collaborations() {
		cj := collaborationJSON{
			Target:        col.Target.Name(),
			Feature:       col.Feature.Name,
			Description:   col.Description,
			Cardinality:   col.Cardinality,
			CardinalityBy: col.CardinalityBy,
			Flow:          col.Flow.Name,
			Step:          col.Step,
			StepOrder:     col.StepOrder,
		}
		if col.DeliveredBy != nil {
			cj.DeliveredBy = col.DeliveredBy.Name()
		}
		collabs = append(collabs, cj)
	}
	j := componentJSON{
		Name:           c.Name(),
		Domain:         string(c.Domain()),
		Kind:           string(c.Kind()),
		Visibility:     string(c.Visibility()),
		Collaborations: collabs,
	}
	if svc, ok := c.(*graph.Service); ok {
		j.Platform = svc.Platform
	}
	if mb, ok := c.(*graph.MessageBroker); ok {
		j.BrokerTechnology = mb.BrokerTechnology
		j.CloudProvider = mb.CloudProvider
	}
	if ev, ok := c.(*graph.Event); ok && ev.MessageBroker() != nil {
		mb := ev.MessageBroker()
		j.PublishedAt = &brokerJSON{
			Name:       mb.Name(),
			Technology: mb.BrokerTechnology,
			Cloud:      mb.CloudProvider,
		}
	}
	return j
}
