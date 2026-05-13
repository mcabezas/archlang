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
)

// HTTPServer serves the architecture graph over HTTP.
type HTTPServer struct {
	storage Storage
	drawer  *mermaidDrawer
	addr    string
}

// NewHTTPServer creates a new HTTP server that serves the architecture graph.
func NewHTTPServer(graphs []*graph.Graph, addr string) *HTTPServer {
	return &HTTPServer{
		storage: New(graphs),
		drawer:  &mermaidDrawer{},
		addr:    addr,
	}
}

// Start begins listening and serving HTTP requests.
// It blocks until an interrupt signal (SIGINT, SIGTERM) is received,
// then gracefully shuts down allowing in-flight requests to complete.
func (s *HTTPServer) Start() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/components/", s.handleGetComponents)
	mux.HandleFunc("/diagram", s.handleDiagram)

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

func (s *HTTPServer) handleDiagram(w http.ResponseWriter, r *http.Request) {
	feature := r.URL.Query().Get("feature")

	w.Header().Set("Content-Type", "text/html")

	if feature != "" {
		components, err := s.storage.FindByFeature(feature)
		if err != nil {
			http.Error(w, "feature not found", http.StatusNotFound)
			return
		}
		fmt.Fprint(w, s.drawer.drawFeature(components, feature))
		return
	}

	all, err := s.storage.ListAll()
	if err != nil {
		http.Error(w, "failed to list components", http.StatusInternalServerError)
		return
	}
	fmt.Fprint(w, s.drawer.draw(all))
}

type componentJSON struct {
	Name           string              `json:"name"`
	Domain         string              `json:"domain,omitempty"`
	Kind           string              `json:"kind"`
	Visibility     string              `json:"visibility"`
	Collaborations []collaborationJSON `json:"collaborations"`
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
}

func toComponentJSON(c graph.Component) componentJSON {
	var collabs []collaborationJSON
	for _, col := range c.Collaborations() {
		collabs = append(collabs, collaborationJSON{
			Target:        col.Target.Name(),
			Feature:       col.Feature.Name,
			Description:   col.Description,
			Cardinality:   col.Cardinality,
			CardinalityBy: col.CardinalityBy,
			Flow:          col.Flow.Name,
			Step:          col.Step,
			StepOrder:     col.StepOrder,
		})
	}
	return componentJSON{
		Name:           c.Name(),
		Domain:         string(c.Domain()),
		Kind:           kindOf(c),
		Visibility:     string(c.Visibility()),
		Collaborations: collabs,
	}
}

func kindOf(c graph.Component) string {
	switch c.(type) {
	case *graph.Service:
		return "service"
	case *graph.Infra:
		return "infra"
	default:
		return "component"
	}
}
